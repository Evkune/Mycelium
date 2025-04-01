// Copyright (c) OpenFaaS Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"strconv"

	"encoding/json"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/openfaas/connector-sdk/types"
	"github.com/openfaas/faas-provider/auth"
)

type StringTriple struct {
	First  string
	Second string
	Third  string
}

var (
	gatewayUsername string
	gatewayPassword string
	gatewayURL      string
	trimChannelKey  bool
	asyncInvoke     bool
	topic           string
	broker          string

	functionTags map[string][]StringTriple // Maps of function_id in keys and tuple of tags and function_name in values
	topicFunctions map[string][]string // Map of topics in keys and functions (Function_id) in values
	synchronized_routers bool // Indicates if the router is synchronized
	creds *auth.BasicAuthCredentials // Credentials for openfaas gateway
)

func getTopicsAndFunctions(url_monitoring string) (map[string][]string, map[string][]StringTriple, bool, error) {
	// Get the topics and functions from the monitoring service
	log.Printf("Getting topics and functions from: %s", url_monitoring)
	resp, err := http.Get(url_monitoring + "/topics-functions")
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		TopicFunctions map[string][]string      `json:"topics"`
		FunctionsTags  map[string][]StringTriple `json:"functions"`
		synchronized_routers bool `json:"synchronized"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, nil, err
	}

	return response.TopicFunctions, response.FunctionsTags, response.synchronized, nil
}

func invokeFunction(functionName, message string) error {
	// Invoke the function with the message
	url := fmt.Sprintf("%s/function/%s", gatewayURL, functionName)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.SetBasicAuth(creds.User, creds.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	return nil
}

func invokingFromTopic(topic string, message string) error {
	// Invokes the greatest tag of all the functions subscribed to the topic
	functions, ok := topicFunctions[topic]
	if !ok {
		log.Printf("No function for topic: %s", topic)
		return nil
	}

	for _, function := range functions {
		functionToInvoke := make([]StringTriple, 0)
		max := 0.0
		for _, tag := range functionTags[function] {
			tagFloat, err := strconv.ParseFloat(tag.First, 64)
			if err != nil {
				fmt.Println("Error converting:", err)
			}
			if (tagFloat > max){
				max = tagFloat
				functionToInvoke = tag
			}
		}
		if functionToInvoke.Third == "2" {
			// Logic to handle when both routers have the function and the same tag
		}
		fmt.Println("Function invoked:", functionToInvoke)
		invokeFunction(functionToInvoke.Second, message)
		
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


	gatewayUsername = os.Getenv("gw-username")
	gatewayPassword = os.Getenv("gw-password")
	gatewayURL = os.Getenv("gw-flag")
	broker = os.Getenv("broker")
	asyncInvoke = false
	trimChannelKey = false

	password := ""
	user := ""
	cleansess := false
	id := "router"
	qos := 0
	topic = "#"

	flag.Parse()

	if len(gatewayPassword) > 0 {
		log.Printf("Trying gateway credentials from env")
		creds = &auth.BasicAuthCredentials{
			User:     gatewayUsername,
			Password: gatewayPassword,
		}
	} else {
		creds = types.GetCredentials()
	}

	contentType := "application/json"
	if v, exists := os.LookupEnv("content_type"); exists && len(v) > 0 {
		contentType = v
	}

	if len(gatewayURL) == 0 {
		log.Panicln(`a value must be set for env "gatewayURL" or via the -gateway flag for your OpenFaaS gateway`)
		return
	}

	config := &types.ControllerConfig{
		RebuildInterval:          time.Millisecond * 1000,
		GatewayURL:               gatewayURL,
		PrintResponse:            true,
		PrintResponseBody:        true,
		TopicAnnotationDelimiter: ",",
		AsyncFunctionInvocation:  asyncInvoke,
		ContentType:              contentType,
	}

	log.Printf("Topic: %q\tBroker: %q\n", topic, broker)
	log.Printf("Gateway: %s\tAsync: %v\n", gatewayURL, asyncInvoke)

	controller := types.NewController(creds, config)

	receiver := ResponseReceiver{}
	controller.Subscribe(&receiver)

	log.Println("Listing deployed functions and their topics:")
	res_map_topics, res_map_functions, res_synchronized, err := getTopicsAndFunctions(os.Getenv("url-monitoring"))
	topicFunctions = res_map_topics
	functionTags = res_map_functions
	synchronized_routers = res_synchronized
	if err != nil {
		log.Printf("Error listing functions: %s", err)
	} else {
		for topic, functions := range topicFunctions {
			log.Printf("Topics: %s, Functions: %s", topic, strings.Join(functions, ", "))
		}
		for functionID, tags := range functionTags {
			log.Printf("Function ID: %s, Tags: %v", functionID, tags)
		}
	}
	controller.BeginMapBuilder()

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

			if trimChannelKey {
				log.Printf("Topic before trim: %s", topic)
				index := strings.Index(topic, "/")
				topic = topic[index+1:]
			}
			
			list_topics := make([]string, 0)
			for key, _ := range topicFunctions {
				list_topics = append(list_topics, key)
			}
			if !contains(list_topics, topic) {
				log.Printf("Topic not found: %s", topic)
				continue
			}

			invokingFromTopic(topic, string(data))
			receiveCount++
		}

		client.Disconnect(1250)
	}()

	select {}
}

// ResponseReceiver enables connector to receive results from the
// function invocation
type ResponseReceiver struct {
}

// Response is triggered by the controller when a message is
// received from the function invocation
func (ResponseReceiver) Response(res types.InvokerResponse) {
	if res.Error != nil {
		log.Printf("tester got error: %s", res.Error.Error())
	} else {
		log.Printf("tester got result: [%d] %s => %s (%d) bytes", res.Status, res.Topic, res.Function, len(*res.Body))
	}
}
