package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"fmt"

	"github.com/openfaas/faas-provider/auth"
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
	map_topicFunctions map[string][]string
	map_functionsTagsCombined  map[string][]FunctionData
	map_functionsTags  map[string][]FunctionTuple
	synchronized_routers bool
	mu                 sync.RWMutex
)

func listTopicsAndFunctions() (map[string][]string, map[string][]FunctionTuple, error) {
	// Function that fetch the topics and functions from the openfaas gateway
	gatewayURL := os.Getenv("gw-url")
	gatewayUsername := os.Getenv("gw-username")
	gatewayPassword := os.Getenv("gw-password")

	creds := &auth.BasicAuthCredentials{
		User:     gatewayUsername,
		Password: gatewayPassword,
	}

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

	topic_func := make(map[string][]string)
	function_tags := make(map[string][]FunctionTuple)

	for _, function := range functions {
		functionName := function["name"].(string) // Name of the function, distinct for every function, used to invoke the function
		if annotations, ok := function["annotations"].(map[string]interface{}); ok {
			if functionId, exists := annotations["functionId"]; exists {
				functionIdStr := functionId.(string) // ID of the function, used to have multiple versions of a same function, ex: a poor version and a good version
				tag := ""
				if tagAnnotation, exists := annotations["tag"]; exists {
					tag = tagAnnotation.(string)
				}
				function_tags[functionIdStr] = append(function_tags[functionIdStr], FunctionTuple{Tag: tag, FunctionName: functionName})
				if topicAnnotation, exists := annotations["topic"]; exists {
					topics := strings.Split(topicAnnotation.(string), ",")
					for _, topic := range topics {
						topic_func[topic] = append(topic_func[topic], functionIdStr)
					}
				}
			}
		}
	}

	// Remove duplicates from topicFunctions
	for topic, functionIDs := range topic_func {
		uniqueFunctionIDs := make(map[string]bool)
		for _, functionID := range functionIDs {
			uniqueFunctionIDs[functionID] = true
		}
		distinctFunctionIDs := make([]string, 0, len(uniqueFunctionIDs))
		for functionID := range uniqueFunctionIDs {
			distinctFunctionIDs = append(distinctFunctionIDs, functionID)
		}
		topic_func[topic] = distinctFunctionIDs
	}

	// Remove duplicates from functionsTags
	for functionID, tuples := range function_tags {
		seenTags := make(map[string]bool)
		uniqueTuples := []FunctionTuple{}

		for _, tuple := range tuples {
			if !seenTags[tuple.Tag] {
				seenTags[tuple.Tag] = true
				uniqueTuples = append(uniqueTuples, tuple)
			}
		}

		// Update the map with the unique tuples
		function_tags[functionID] = uniqueTuples
	}
	return topic_func, function_tags, nil
}

func getOtherRouterFunctions() (map[string][]string, map[string][]FunctionTuple, error) {
	other_monitoring_service := os.Getenv("other-monitoring-service")
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, other_monitoring_service+"/monitoring-functions", nil)
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
			fmt.Println("Error fetching this router functions: %s", err)
		}
		newOtherTopic, newOtherFunction, err := getOtherRouterFunctions()
		if err != nil {
			fmt.Println("Error fetching other router functions: %s", err)
			mu.Lock()
			synchronized_routers = false
			mu.Unlock()
		}else {
			mu.Lock()
			synchronized_routers = true
			mu.Unlock()
		}
		combinedTopicFunctions := mergeTopicsMaps(newTopicFunctions, newOtherTopic)
		combinedFunctionsTags := mergeAndTransformFunctionsMaps(newFunctionsTags, newOtherFunction)
		mu.Lock()
		map_topicFunctions = combinedTopicFunctions
		map_functionsTagsCombined = combinedFunctionsTags
		map_functionsTags = newFunctionsTags
		mu.Unlock()
		
		fmt.Println("Updated topics and functions: %d topics, %d functions", len(map_topicFunctions), len(map_functionsTags))
		/*
		for topic, functions := range map_topicFunctions {
			fmt.Println("Topic: %s, Functions: %v", topic, functions)
		}
		for functionId, tuples := range map_functionsTags {
			fmt.Println("Function ID: %s, Tags: %v", functionId, tuples)
		}
		fmt.Println("Synchronized with other router: %t", synchronized_routers)
		*/
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
	/*
	fmt.Println("Map 1:")
	for functionId, tuples := range map1 {
		fmt.Println("Function ID: %s, Tags: %v", functionId, tuples)
	}
	fmt.Println("Map 2:")
	for functionId, tuples := range map2 {
		fmt.Println("Function ID: %s, Tags: %v", functionId, tuples)
	}*/
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
	for functionId, tuples := range result {
		fmt.Println("Function ID: %s, Tags: %v", functionId, tuples)
	}
    return result
} 

func handler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topics":    map_topicFunctions,
		"functions": map_functionsTagsCombined,
		"synchronized": synchronized_routers,
	})
}

func handler2(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topics":    map_topicFunctions,
		"functions": map_functionsTags,
	})
}

func main() {
	go updateTopicsAndFunctions()

	http.HandleFunc("/topics-functions", handler)
	http.HandleFunc("/monitoring-functions", handler2)
	port := os.Getenv("port")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Starting server on port %s", port)
	fmt.Println(http.ListenAndServe(":"+port, nil))
}
