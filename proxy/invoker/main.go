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
	"log"
	"github.com/openfaas/faas-provider/auth"
)

type FailedInvokation struct {
	Function string
	Message  string
	InvokingNumber int
}

var (
	gatewayURL string
	creds *auth.BasicAuthCredentials

	failedMessages map[int]FailedInvokation
	uniqueID int
	mu sync.RWMutex	
)

func invokeFunction(functionName, message string) error {
	// Invoke the function with the message
	url := fmt.Sprintf("%s/function/%s", gatewayURL, functionName)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(message))
	log.Printf("Invoking function: %s", functionName)
	if err != nil {
		log.Printf("Error creating request: %s", err)
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
		var toDelete []int // Track IDs to delete after iteration
		for id, failedInvokation := range failedMessages {
			err := invokeFunction(failedInvokation.Function, failedInvokation.Message)
			if err != nil {
				log.Printf("Failed to invoke function %s: %v", failedInvokation.Function, err)
				
				failedInvokation.InvokingNumber++
				if failedInvokation.InvokingNumber > 5 {
					log.Printf("Failed to invoke function %s after 5 attempts", failedInvokation.Function)
                    toDelete = append(toDelete, id) // Mark for deletion
				} else {
					failedMessages[id] = failedInvokation
				}
			} else {
				log.Printf("Successfully invoked function %s", failedInvokation.Function)
                toDelete = append(toDelete, id) // Mark for deletion
			}
		}
		
		// Delete marked entries after iteration
		for _, id := range toDelete {
			delete(failedMessages, id)
		}
		length := len(failedMessages)
		mu.Unlock()
		time.Sleep(time.Duration((length+1)*30) * time.Second)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("Error invoking function: %v", err) // Debug log
		http.Error(w, fmt.Sprintf("Failed to invoke function: %v", err), http.StatusInternalServerError)
		// Store the failed message for later retry
		failedInvokation := FailedInvokation{
			Function: functionToInvoke,
			Message:  message,
			InvokingNumber: 1,
		}
		mu.Lock()
		failedMessages[uniqueID] = failedInvokation
		uniqueID++
		mu.Unlock()
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
	gatewayUsername := os.Getenv("gw-username")
	gatewayPassword := os.Getenv("gw-password")

	uniqueID = 0
	failedMessages = make(map[int]FailedInvokation)

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
	log.Printf("Starting server on port %s", port)
	log.Println(http.ListenAndServe(":"+port, nil))
}
