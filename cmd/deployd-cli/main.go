package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// CLI client for go-deployd API
type CLI struct {
	baseURL string
	token   string
	client  *http.Client
}

// AuthResponse represents the login response
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
}

func main() {
	var (
		host      = flag.String("host", "http://localhost:2403", "go-deployd server URL")
		masterKey = flag.String("master-key", "", "master key for authentication")
		username  = flag.String("username", "", "username for user authentication")
		password  = flag.String("password", "", "password for user authentication")
		command   = flag.String("cmd", "", "command to execute (login, get, post, put, delete)")
		resource  = flag.String("resource", "", "resource/collection name")
		id        = flag.String("id", "", "resource ID (for get, put, delete)")
		data      = flag.String("data", "", "JSON data (for post, put)")
	)
	flag.Parse()

	cli := &CLI{
		baseURL: *host,
		client:  &http.Client{},
	}

	// Handle commands
	switch *command {
	case "login":
		if *masterKey != "" {
			// Master key authentication
			if err := cli.loginWithMasterKey(*masterKey); err != nil {
				fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
				os.Exit(1)
			}
		} else if *username != "" && *password != "" {
			// User/password authentication
			if err := cli.loginWithUserPassword(*username, *password); err != nil {
				fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: Either master-key or username+password required for login\n")
			os.Exit(1)
		}
		fmt.Println("Login successful!")

	case "get":
		if *resource == "" {
			fmt.Fprintf(os.Stderr, "Error: resource required\n")
			os.Exit(1)
		}
		cli.loadToken()
		path := fmt.Sprintf("/%s", *resource)
		if *id != "" {
			path = fmt.Sprintf("/%s/%s", *resource, *id)
		}
		if err := cli.request("GET", path, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
			os.Exit(1)
		}

	case "post":
		if *resource == "" || *data == "" {
			fmt.Fprintf(os.Stderr, "Error: resource and data required\n")
			os.Exit(1)
		}
		cli.loadToken()
		if err := cli.request("POST", fmt.Sprintf("/%s", *resource), strings.NewReader(*data)); err != nil {
			fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
			os.Exit(1)
		}

	case "put":
		if *resource == "" || *id == "" || *data == "" {
			fmt.Fprintf(os.Stderr, "Error: resource, id, and data required\n")
			os.Exit(1)
		}
		cli.loadToken()
		if err := cli.request("PUT", fmt.Sprintf("/%s/%s", *resource, *id), strings.NewReader(*data)); err != nil {
			fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
			os.Exit(1)
		}

	case "delete":
		if *resource == "" || *id == "" {
			fmt.Fprintf(os.Stderr, "Error: resource and id required\n")
			os.Exit(1)
		}
		cli.loadToken()
		if err := cli.request("DELETE", fmt.Sprintf("/%s/%s", *resource, *id), nil); err != nil {
			fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Usage: deployd-cli -cmd=<command> [options]\n")
		fmt.Fprintf(os.Stderr, "Commands: login, get, post, put, delete\n")
		fmt.Fprintf(os.Stderr, "\nAuthentication:\n")
		fmt.Fprintf(os.Stderr, "  Login with master key: -cmd=login -master-key=<key>\n")
		fmt.Fprintf(os.Stderr, "  Login with user/pass:  -cmd=login -username=<user> -password=<pass>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func (c *CLI) loginWithMasterKey(masterKey string) error {
	payload := map[string]string{"masterKey": masterKey}
	data, _ := json.Marshal(payload)

	resp, err := c.client.Post(c.baseURL+"/auth/login", "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", body)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	// Save token to file
	tokenFile := os.ExpandEnv("$HOME/.deployd-token")
	return os.WriteFile(tokenFile, []byte(authResp.Token), 0600)
}

func (c *CLI) loginWithUserPassword(username, password string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	data, _ := json.Marshal(payload)

	resp, err := c.client.Post(c.baseURL+"/auth/login", "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", body)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	// Save token to file
	tokenFile := os.ExpandEnv("$HOME/.deployd-token")
	return os.WriteFile(tokenFile, []byte(authResp.Token), 0600)
}

func (c *CLI) loadToken() {
	tokenFile := os.ExpandEnv("$HOME/.deployd-token")
	data, err := os.ReadFile(tokenFile)
	if err == nil {
		c.token = string(data)
	}
}

func (c *CLI) request(method, path string, body io.Reader) error {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Pretty print JSON response
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
