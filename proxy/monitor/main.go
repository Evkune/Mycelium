package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"log"
    "context"
    "fmt"

	"github.com/openfaas/faas-provider/auth"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/resource"
)

type FunctionTuple struct { // Same as FunctionData but used only to share to the other monitoring service
	Tag  string
	FunctionName string
}

type FunctionData struct {
	Tag  string // Tag of the function, used to have multiple versions of a same function, ex: a poor version and a good version
	FunctionName string // Name of the function, distinct for every function, used to invoke the function
	Presence  string // 0 if present in this router, 1 if present in other router, 2 if present in both routers
}

var (
	gatewayURL string
	creds *auth.BasicAuthCredentials
	
	otherMonitoringURL string

	topicFunctionsMap map[string][]string
	functionsTagsCombinedMap  map[string][]FunctionData
	functionTagsMap  map[string][]FunctionTuple
	synchronizedProxys bool
	mu                 sync.RWMutex
)

func listTopicsAndFunctions() (map[string][]string, map[string][]FunctionTuple, error) {
	// Function that fetch the topics and functions from the openfaas gateway
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, gatewayURL+"/system/functions", nil)
	if err != nil {
		return nil, nil, err
	}

	req.SetBasicAuth(creds.User, creds.Password)

	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	var functions []map[string]interface{}
	if err := json.Unmarshal(body, &functions); err != nil {
		return nil, nil, err
	}

	topicFunctions := make(map[string][]string)
	functionTags := make(map[string][]FunctionTuple)

	for _, function := range functions {
		functionName := function["name"].(string) // Name of the function, distinct for every function, used to invoke the function
		if annotations, ok := function["annotations"].(map[string]interface{}); ok {
			if functionId, exists := annotations["functionId"]; exists {
				functionIdStr := functionId.(string) // ID of the function, used to have multiple versions of a same function, ex: a poor version and a good version
				tag := ""
				if tagAnnotation, exists := annotations["tag"]; exists {
					tag = tagAnnotation.(string)
				}
				functionTags[functionIdStr] = append(functionTags[functionIdStr], FunctionTuple{Tag: tag, FunctionName: functionName})
				if topicAnnotation, exists := annotations["topic"]; exists {
					topics := strings.Split(topicAnnotation.(string), ",")
					for _, topic := range topics {
						topicFunctions[topic] = append(topicFunctions[topic], functionIdStr)
					}
				}
			}
		}
	}

	// Remove duplicates from topicFunctions
	for topic, functionIDs := range topicFunctions {
		uniqueFunctionIDs := make(map[string]bool)
		for _, functionID := range functionIDs {
			uniqueFunctionIDs[functionID] = true
		}
		distinctFunctionIDs := make([]string, 0, len(uniqueFunctionIDs))
		for functionID := range uniqueFunctionIDs {
			distinctFunctionIDs = append(distinctFunctionIDs, functionID)
		}
		topicFunctions[topic] = distinctFunctionIDs
	}

	// Remove duplicates from functionsTags
	for functionID, tuples := range functionTags {
		seenTags := make(map[string]bool)
		uniqueTuples := []FunctionTuple{}

		for _, tuple := range tuples {
			if !seenTags[tuple.Tag] {
				seenTags[tuple.Tag] = true
				uniqueTuples = append(uniqueTuples, tuple)
			}
		}

		// Update the map with the unique tuples
		functionTags[functionID] = uniqueTuples
	}
	return topicFunctions, functionTags, nil
}

func getOtherRouterFunctions() (map[string][]string, map[string][]FunctionTuple, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, otherMonitoringURL+"/monitoring-functions", nil)
	if err != nil {
		return nil, nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		TopicFunctions map[string][]string      `json:"topics"`
		FunctionsTags  map[string][]FunctionTuple `json:"functions"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, nil, err
	}
	return response.TopicFunctions, response.FunctionsTags, nil
}

func updateTopicsAndFunctions() {
	for {
		newTopicFunctions, newFunctionsTags, err := listTopicsAndFunctions()
		if err != nil {
			log.Printf("Error fetching this router functions: %s", err)
		}
		newOtherTopic, newOtherFunction, err := getOtherRouterFunctions()
		if err != nil {
			log.Printf("Error fetching other router functions: %s", err)
			mu.Lock()
			synchronizedProxys = false
			mu.Unlock()
		}else {
			mu.Lock()
			synchronizedProxys = true
			mu.Unlock()
		}
		combinedTopicFunctions := mergeTopicsMaps(newTopicFunctions, newOtherTopic)
		combinedFunctionsTags := mergeAndTransformFunctionsMaps(newFunctionsTags, newOtherFunction)
		mu.Lock()
		topicFunctionsMap = combinedTopicFunctions
		functionsTagsCombinedMap = combinedFunctionsTags
		functionTagsMap = newFunctionsTags
		mu.Unlock()
		time.Sleep(30 * time.Second)
	}
}

func mergeTopicsMaps(map1, map2 map[string][]string) map[string][]string {
	for k, v := range map2 {
		if _, exists := map1[k]; !exists {
			map1[k] = v
		} else {
			map1[k] = append(map1[k], v...)
		}
	}
	// Remove duplicates from map1
	for k, v := range map1 {
		seen := make(map[string]bool)
		unique := []string{}
		for _, function := range v {
			if !seen[function] {
				seen[function] = true
				unique = append(unique, function)
			}
		}
		map1[k] = unique
	}
	return map1
}

func mergeAndTransformFunctionsMaps(map1, map2 map[string][]FunctionTuple) map[string][]FunctionData {
    result := make(map[string][]FunctionData)
    // Add all entries from map1
    for functionId, tuples := range map1 {
        for _, tuple := range tuples {
            // Initialize the result map with triplets from map1
            result[functionId] = append(result[functionId], FunctionData{
                Tag:  tuple.Tag,  // Tag
                FunctionName: tuple.FunctionName, // Function name
                Presence:  "0",          // Present only in map1
            })
        }
    }

    // Add all entries from map2
    for functionId, tuples := range map2 {
        for _, tuple := range tuples {
            found := false
            // Check if the same functionId and tag exist in map1
            for i, existing := range result[functionId] {
                if existing.Tag == tuple.Tag { // Compare tags
                    // Update the third field to 2 if present in both maps
                    result[functionId][i].Presence = "2"
                    found = true
                    break
                }
            }
            if !found {
                // Add new triplet for entries only in map2
                result[functionId] = append(result[functionId], FunctionData{
                    Tag:  tuple.Tag,  // Tag
                    FunctionName: tuple.FunctionName, // Function name
                    Presence:  "1",          // Present only in map2
                })
            }
        }
    }
    return result
} 

func getMetrics() (float64, float64, error) {
    config, err := rest.InClusterConfig()
    if err != nil {
		return 0, 0, fmt.Errorf("Failed to create in-cluster config: %v", err)
	}

    metricsClient, err := metricsclient.NewForConfig(config)
    if err != nil {
		return 0, 0, fmt.Errorf("Failed to create metrics client: %v", err)
    }

    k8sClient, err := kubernetes.NewForConfig(config)
    if err != nil {
		return 0, 0, fmt.Errorf("Failed to create Kubernetes client: %v", err)
    }

    // Get usage
    nodeMetricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
    if err != nil {
		return 0, 0, fmt.Errorf("Failed to list node metrics: %v", err)
    }

    totalCPUUsed := resource.NewQuantity(0, resource.DecimalSI)
    totalMemUsed := resource.NewQuantity(0, resource.BinarySI)
    for _, node := range nodeMetricsList.Items {
        totalCPUUsed.Add(*node.Usage.Cpu())
        totalMemUsed.Add(*node.Usage.Memory())
    }

    // Get allocatable capacity
    nodes, err := k8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return 0, 0, fmt.Errorf("Failed to list nodes: %v", err)
    }

    totalCPUAlloc := resource.NewQuantity(0, resource.DecimalSI)
    totalMemAlloc := resource.NewQuantity(0, resource.BinarySI)
    for _, node := range nodes.Items {
        totalCPUAlloc.Add(*node.Status.Allocatable.Cpu())
        totalMemAlloc.Add(*node.Status.Allocatable.Memory())
    }

    cpuUsagePercent := float64(totalCPUUsed.MilliValue()) / float64(totalCPUAlloc.MilliValue()) * 100
    memUsagePercent := float64(totalMemUsed.Value()) / float64(totalMemAlloc.Value()) * 100
	return cpuUsagePercent, memUsagePercent, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	cpuUsagePercent, memUsagePercent, err := getMetrics()
	if err != nil {
		http.Error(w, "Error getting metrics", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topics":    topicFunctionsMap,
		"functions": functionsTagsCombinedMap,
		"synchronized": synchronizedProxys,
		"cpu": cpuUsagePercent,
		"memory": memUsagePercent,
	})
}

func handler2(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topics":    topicFunctionsMap,
		"functions": functionTagsMap,
	})
}

func main() {
	gatewayURL = os.Getenv("gw-url")
	gatewayUsername := os.Getenv("gw-username")
	gatewayPassword := os.Getenv("gw-password")

	creds = &auth.BasicAuthCredentials{
		User:     gatewayUsername,
		Password: gatewayPassword,
	}

	otherMonitoringURL = os.Getenv("other-monitoring-service")

	go updateTopicsAndFunctions()

	http.HandleFunc("/topics-functions", handler)
	http.HandleFunc("/monitoring-functions", handler2)
	port := os.Getenv("port")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	log.Println(http.ListenAndServe(":"+port, nil))
}
