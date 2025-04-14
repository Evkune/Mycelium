// Copyright (c) OpenFaaS Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"fmt"
	"log"
	"strings"
	"strconv"
	"sync"
	"io"
	"os"
	"time"
	"net/http"
	"encoding/json"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type FunctionData struct {
	Tag  string
	FunctionName string
	Presence  string 
}

var (
	monitoringURL string
	invokerURL string
	otherInvokerURL string

	topic           string
	broker          string

	functionTags map[string][]FunctionData // Maps of function_id in keys and tuple of tags and function_name in values
	topicFunctions map[string][]string // Map of topics in keys and functions (Function_id) in values
	synchronized_routers bool // Indicates if the router is synchronized
	cpuUsage float64 // CPU usage of the router
	memoryUsage float64 // Memory usage of the router
	mu sync.RWMutex
	once sync.Once
)

func getTopicsAndFunctions() {
	for {
		time.Sleep(30 * time.Second)
		// Get the topics and functions from the monitoring service
		log.Printf("Getting topics and functions from: %s", monitoringURL)
		resp, err := http.Get(monitoringURL + "/topics-functions")
		if err != nil {
			log.Printf("Error getting topics and functions: %s", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %s", err)
			continue
		}

		var response struct {
			TopicFunctions map[string][]string      `json:"topics"`
			FunctionsTags  map[string][]FunctionData `json:"functions"`
			SynchronizedRouters bool `json:"synchronized"`
			CPUUsage float64 `json:"cpu"`
			MemoryUsage float64 `json:"memory"`
		}

		err = json.Unmarshal(body, &response)
		if  err != nil {
			log.Printf("Error unmarshalling JSON: %s \n", err)
			continue
		}
		mu.Lock()
		synchronized_routers = response.SynchronizedRouters
		functionTags = response.FunctionsTags
		topicFunctions = response.TopicFunctions
		cpuUsage = response.CPUUsage
		memoryUsage = response.MemoryUsage
		mu.Unlock()

        // Ensure this block runs only once
        once.Do(func() {
            log.Printf("First iteration of getTopicsAndFunctions completed")
            for topic, functions := range topicFunctions {
                log.Printf("Topics: %s, Functions: %s", topic, strings.Join(functions, ", "))
            }
            for functionID, tags := range functionTags {
                log.Printf("Function ID: %s, Tags: %v", functionID, tags)
            }
			log.Printf("Synchronized Routers: %t", synchronized_routers)
			log.Printf("CPU Usage: %f", cpuUsage)
			log.Printf("Memory Usage: %f", memoryUsage)
        })
	}
}

func postInvocation(functionName string, message string, invoker int) error {
	// Post the invocation to the function
	url := ""
	if invoker == 0 {
		url = fmt.Sprintf("%s/invoke?function=%s", invokerURL, functionName)
	} else {
		url = fmt.Sprintf("%s/invoke?function=%s", otherInvokerURL, functionName)
	}
	log.Printf("Posting invocation to: %s", url)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Error sending function %s: %s", functionName, res.Status)
	}
	return nil
}

func postInvocationWithBackoff(functionName string, message string, invoker int) {
    backoff := 2 * time.Second
    for i := 0; i < 5; i++ {
        err := postInvocation(functionName, message, invoker)
        if err == nil {
            return
        }
        time.Sleep(backoff)
        backoff *= 2 // Backoff exponentiel
    }
	log.Printf("Failed to invoke function %s after retries", functionName)
}

func routing(topic string, message string) error {
	mu.RLock()
	defer mu.RUnlock()
	// Invokes the greatest tag of all the functions subscribed to the topic
	functions, ok := topicFunctions[topic]
	if !ok {
		log.Printf("No function for topic: %s", topic)
		return nil
	}

	for _, function := range functions {
		functionToInvoke := FunctionData{
			Tag: "0.0",
			FunctionName: "",
			Presence: "0",
		}
		max := 0.0
		for _, functionTuple := range functionTags[function] {
			tagFloat, err := strconv.ParseFloat(functionTuple.Tag, 64)
			if err != nil {
				log.Printf("Error converting tag to float: %s", functionTuple.Tag)
			}
			if (tagFloat > max){
				max = tagFloat
				functionToInvoke.FunctionName = functionTuple.FunctionName
				functionToInvoke.Tag = functionTuple.Tag
				functionToInvoke.Presence = functionTuple.Presence
			}
		}
		if functionToInvoke.Presence == "2" {
			// Logic to handle when both routers have the function and the same tag
			if cpuUsage > 80 ||  memoryUsage > 80 {
				log.Printf("Cluster overloaded, invoking function: %s with tag: %s", functionToInvoke.FunctionName, functionToInvoke.Tag)
				postInvocationWithBackoff(functionToInvoke.FunctionName, message, 1)
			} else {
				randomInt := int(time.Now().UnixNano() % 2)
				postInvocationWithBackoff(functionToInvoke.FunctionName, message, randomInt)
			}
		} else if functionToInvoke.Presence == "1" {
			postInvocationWithBackoff(functionToInvoke.FunctionName, message, 1)
		} else {
			postInvocationWithBackoff(functionToInvoke.FunctionName, message, 0)
		}
		log.Printf("Invoking function: %s with tag: %s", functionToInvoke.FunctionName, functionToInvoke.Tag)
	}
	return nil
}


func contains(slice []string, value string) bool {
	// Utility function to check if a string is in a slice (used for topics)
    for _, v := range slice {
        if v == value {
            return true
        }
    }
    return false
}

func main() {
	monitoringURL = os.Getenv("url-monitoring")
	invokerURL = os.Getenv("url-invoker")
	otherInvokerURL = os.Getenv("url-other-invoker")

	broker = os.Getenv("broker")

	password := ""
	user := ""
	cleansess := false
	id := os.Getenv("client-id")
	qos := 0
	topic = "#"

	go getTopicsAndFunctions()
	
	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(id)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetCleanSession(cleansess)

	receiveCount := 0
	msgCh := make(chan [2]string)

	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		log.Printf("Message incoming")
		msgCh <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	opts.SetOnConnectHandler(func(client MQTT.Client) {
		log.Printf("Connected to %s", broker)

		if token := client.Subscribe(topic, byte(qos), nil); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
			os.Exit(1)
		}
		log.Printf("Subscribed to topic: %s", topic)
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	log.Printf("Connection requested for broker: %s", broker)
	// When a message is received, invoke the function
	go func() {
		for {
			incoming := <-msgCh

			topic := incoming[0]
			data := []byte(incoming[1])
			
			topicsList := make([]string, 0)
			for key, _ := range topicFunctions {
				topicsList = append(topicsList, key)
			}
			if !contains(topicsList, topic) {
				log.Printf("Topic not found: %s", topic)
				continue
			}

			routing(topic, string(data))
			receiveCount++
		}

		client.Disconnect(1250)
	}()

	select {}
}
