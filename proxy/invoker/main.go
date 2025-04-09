package main

import (
	"encoding/json"
	"net/http"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"fmt"
	"github.com/openfaas/faas-provider/auth"
)

type FailedInvokation struct {
	Function string
	Message  string
	InvokingNumber int
}

var (
	failedMessages map[int]FailedInvokation
	gatewayURL string
	gatewayUsername string
	gatewayPassword string
	
	uniqueID int
	mu sync.RWMutex

	creds *auth.BasicAuthCredentials
	
)

func invokeFunction(functionName, message string) error {
	// Invoke the function with the message
	url := fmt.Sprintf("%s/function/%s", gatewayURL, functionName)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(message))
	fmt.Println("Invoking function:", functionName)
	if err != nil {
		fmt.Println("Error creating request:", err)
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

func invokingFailedFunctions() {
	for {
		mu.Lock()
		for id, failedInvokation := range failedMessages {
			err := invokeFunction(failedInvokation.Function, failedInvokation.Message)
			if err != nil {
				fmt.Printf("Failed to invoke function %s: %v\n", failedInvokation.Function, err)
				
				failedInvokation.InvokingNumber++
				if failedInvokation.InvokingNumber > 5 {

					fmt.Printf("Failed to invoke function %s after 5 attempts\n", failedInvokation.Function)
					delete(failedMessages, id)
					continue
				}
			} else {
				fmt.Printf("Successfully invoked function %s\n", failedInvokation.Function)
				delete(failedMessages, id)
			}
		}
		mu.Unlock()
		time.Sleep(30*time.Second)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	functionToInvoke := r.URL.Query().Get("function")
	if functionToInvoke == "" {
		http.Error(w, "Function name is required", http.StatusBadRequest)
		return
	}

    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close() // Ensure the body is closed after reading

    message := string(body) // Convert the body to a string

	err = invokeFunction(functionToInvoke, message)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to invoke function: %v", err), http.StatusInternalServerError)
		// Store the failed message for later retry
		mu.Lock()
		defer mu.Unlock()
		failedInvokation := FailedInvokation{
			Function: functionToInvoke,
			Message:  message,
			InvokingNumber: 1,
		}

		failedMessages[uniqueID] = failedInvokation
		uniqueID++
		return
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Function invoked successfully",
		})
	}

}

func main() {
	gatewayURL = os.Getenv("gw-url")
	gatewayUsername = os.Getenv("gw-username")
	gatewayPassword = os.Getenv("gw-password")

	uniqueID = 0

	creds = &auth.BasicAuthCredentials{
		User:     gatewayUsername,
		Password: gatewayPassword,
	}

	go invokingFailedFunctions()

	http.HandleFunc("/invoke", handler)
	port := os.Getenv("port")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Starting server on port %s", port)
	fmt.Println(http.ListenAndServe(":"+port, nil))
}
