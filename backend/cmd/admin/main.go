package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sgerhart/aegisflux/backend/services/registry/signing"
	"github.com/urfave/cli/v2"
)

// CLI application for AegisFlux backend administration
func main() {
	app := &cli.App{
		Name:    "aegisflux-admin",
		Usage:   "Administrative CLI for AegisFlux backend services",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "Key management operations",
				Subcommands: []*cli.Command{
					{
						Name:  "rotate",
						Usage: "Rotate signing keys",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "keys-path",
								Usage:    "Path to signing keys file",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "backup-path",
								Usage: "Path to backup keys before rotation",
							},
						},
						Action: rotateKeys,
					},
					{
						Name:  "list",
						Usage: "List available signing keys",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "keys-path",
								Usage:    "Path to signing keys file",
								Required: true,
							},
						},
						Action: listKeys,
					},
					{
						Name:  "backup",
						Usage: "Backup signing keys",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "keys-path",
								Usage:    "Path to signing keys file",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "backup-path",
								Usage:    "Path to backup file",
								Required: true,
							},
						},
						Action: backupKeys,
					},
				},
			},
			{
				Name:    "assignment",
				Aliases: []string{"a"},
				Usage:   "Assignment management operations",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create a new bundle assignment",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "registry-url",
								Usage:    "Registry service URL",
								Value:    "http://localhost:8090",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "bundle-id",
								Usage:    "Bundle ID to assign",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "host-selector",
								Usage:    "Host selector JSON",
								Required: true,
							},
							&cli.IntFlag{
								Name:  "ttl-seconds",
								Usage: "TTL in seconds (optional)",
							},
							&cli.BoolFlag{
								Name:  "dry-run",
								Usage: "Create as dry-run assignment",
							},
							&cli.StringFlag{
								Name:  "created-by",
								Usage: "User creating the assignment",
								Value: "admin-cli",
							},
						},
						Action: createAssignment,
					},
					{
						Name:  "list",
						Usage: "List assignments",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "registry-url",
								Usage: "Registry service URL",
								Value: "http://localhost:8090",
							},
							&cli.IntFlag{
								Name:  "limit",
								Usage: "Limit number of results",
								Value: 50,
							},
							&cli.IntFlag{
								Name:  "offset",
								Usage: "Offset for pagination",
								Value: 0,
							},
						},
						Action: listAssignments,
					},
					{
						Name:  "cancel",
						Usage: "Cancel an assignment",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "registry-url",
								Usage:    "Registry service URL",
								Value:    "http://localhost:8090",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "assignment-id",
								Usage:    "Assignment ID to cancel",
								Required: true,
							},
						},
						Action: cancelAssignment,
					},
				},
			},
			{
				Name:    "bundle",
				Aliases: []string{"b"},
				Usage:   "Bundle management operations",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create a new bundle",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "registry-url",
								Usage:    "Registry service URL",
								Value:    "http://localhost:8090",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Bundle name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "content",
								Usage:    "Bundle content (base64 encoded)",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "description",
								Usage: "Bundle description",
							},
							&cli.StringFlag{
								Name:  "version",
								Usage: "Bundle version",
								Value: "1.0.0",
							},
							&cli.StringFlag{
								Name:  "created-by",
								Usage: "User creating the bundle",
								Value: "admin-cli",
							},
						},
						Action: createBundle,
					},
					{
						Name:  "list",
						Usage: "List bundles",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "registry-url",
								Usage: "Registry service URL",
								Value: "http://localhost:8090",
							},
							&cli.IntFlag{
								Name:  "limit",
								Usage: "Limit number of results",
								Value: 50,
							},
							&cli.IntFlag{
								Name:  "offset",
								Usage: "Offset for pagination",
								Value: 0,
							},
						},
						Action: listBundles,
					},
				},
			},
			{
				Name:    "health",
				Aliases: []string{"h"},
				Usage:   "Health check operations",
				Subcommands: []*cli.Command{
					{
						Name:  "check",
						Usage: "Check service health",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "service-url",
								Usage:    "Service URL to check",
								Required: true,
							},
						},
						Action: checkHealth,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Key management functions

func rotateKeys(c *cli.Context) error {
	keysPath := c.String("keys-path")
	backupPath := c.String("backup-path")

	// Create signer instance
	signer, err := signing.NewSigner(keysPath)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Backup keys if backup path provided
	if backupPath != "" {
		fmt.Printf("Backing up keys to %s...\n", backupPath)
		if err := signer.BackupKeys(backupPath); err != nil {
			return fmt.Errorf("failed to backup keys: %w", err)
		}
		fmt.Println("Keys backed up successfully")
	}

	// Rotate keys
	fmt.Println("Rotating signing keys...")
	if err := signer.RotateKey(); err != nil {
		return fmt.Errorf("failed to rotate keys: %w", err)
	}

	fmt.Println("Keys rotated successfully")
	fmt.Println("Note: You may need to activate the new key using 'key activate' command")

	return nil
}

func listKeys(c *cli.Context) error {
	keysPath := c.String("keys-path")

	signer, err := signing.NewSigner(keysPath)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	keys := signer.ListKeys()
	
	fmt.Printf("Available signing keys (%d total):\n", len(keys))
	fmt.Println("=====================================")
	
	for _, key := range keys {
		fmt.Printf("Key ID: %s\n", key.Kid)
		fmt.Printf("  Algorithm: %s\n", key.Algorithm)
		fmt.Printf("  Created: %s\n", key.CreatedAt.Format(time.RFC3339))
		if !key.ExpiresAt.IsZero() {
			fmt.Printf("  Expires: %s\n", key.ExpiresAt.Format(time.RFC3339))
		}
		fmt.Printf("  Public Key: %s...\n", key.PublicKey[:32])
		fmt.Println()
	}

	return nil
}

func backupKeys(c *cli.Context) error {
	keysPath := c.String("keys-path")
	backupPath := c.String("backup-path")

	signer, err := signing.NewSigner(keysPath)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	fmt.Printf("Backing up keys to %s...\n", backupPath)
	if err := signer.BackupKeys(backupPath); err != nil {
		return fmt.Errorf("failed to backup keys: %w", err)
	}

	fmt.Println("Keys backed up successfully")
	return nil
}

// Assignment management functions

func createAssignment(c *cli.Context) error {
	registryURL := c.String("registry-url")
	bundleIDStr := c.String("bundle-id")
	hostSelectorStr := c.String("host-selector")
	ttlSeconds := c.Int("ttl-seconds")
	dryRun := c.Bool("dry-run")
	createdBy := c.String("created-by")

	// Parse bundle ID
	bundleID, err := uuid.Parse(bundleIDStr)
	if err != nil {
		return fmt.Errorf("invalid bundle ID: %w", err)
	}

	// Parse host selector JSON
	var hostSelector json.RawMessage
	if err := json.Unmarshal([]byte(hostSelectorStr), &hostSelector); err != nil {
		return fmt.Errorf("invalid host selector JSON: %w", err)
	}

	// Prepare request
	req := map[string]interface{}{
		"host_selector": hostSelector,
		"bundle_id":     bundleID,
		"dry_run":       dryRun,
		"created_by":    createdBy,
	}

	if ttlSeconds > 0 {
		req["ttl_seconds"] = ttlSeconds
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(registryURL+"/assignments", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create assignment: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("assignment creation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var assignment map[string]interface{}
	if err := json.Unmarshal(respBody, &assignment); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Assignment created successfully:\n")
	fmt.Printf("  ID: %s\n", assignment["id"])
	fmt.Printf("  Bundle ID: %s\n", assignment["bundle_id"])
	fmt.Printf("  Dry Run: %v\n", assignment["dry_run"])
	if assignment["ttl_ts"] != nil {
		fmt.Printf("  TTL: %s\n", assignment["ttl_ts"])
	}

	return nil
}

func listAssignments(c *cli.Context) error {
	registryURL := c.String("registry-url")
	limit := c.Int("limit")
	offset := c.Int("offset")

	url := fmt.Sprintf("%s/assignments?limit=%d&offset=%d", registryURL, limit, offset)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to list assignments: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to list assignments (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	assignments := result["assignments"].([]interface{})
	fmt.Printf("Assignments (%d total):\n", len(assignments))
	fmt.Println("=====================================")

	for _, assignment := range assignments {
		assign := assignment.(map[string]interface{})
		fmt.Printf("ID: %s\n", assign["id"])
		fmt.Printf("  Bundle ID: %s\n", assign["bundle_id"])
		fmt.Printf("  Status: %s\n", assign["status"])
		fmt.Printf("  Dry Run: %v\n", assign["dry_run"])
		fmt.Printf("  Created By: %s\n", assign["created_by"])
		if assign["ttl_ts"] != nil {
			fmt.Printf("  TTL: %s\n", assign["ttl_ts"])
		}
		fmt.Printf("  Created: %s\n", assign["created_at"])
		fmt.Println()
	}

	return nil
}

func cancelAssignment(c *cli.Context) error {
	registryURL := c.String("registry-url")
	assignmentID := c.String("assignment-id")

	url := fmt.Sprintf("%s/assignments/%s", registryURL, assignmentID)
	
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to cancel assignment: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("assignment cancellation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Assignment cancelled successfully:\n")
	fmt.Printf("  ID: %s\n", result["assignment_id"])
	fmt.Printf("  Cancelled By: %s\n", result["cancelled_by"])
	fmt.Printf("  Cancelled At: %s\n", result["cancelled_at"])

	return nil
}

// Bundle management functions

func createBundle(c *cli.Context) error {
	registryURL := c.String("registry-url")
	name := c.String("name")
	content := c.String("content")
	description := c.String("description")
	version := c.String("version")
	createdBy := c.String("created-by")

	req := map[string]interface{}{
		"name":       name,
		"content":    content,
		"created_by": createdBy,
	}

	if description != "" {
		req["description"] = description
	}
	if version != "" {
		req["version"] = version
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(registryURL+"/bundles", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("bundle creation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var bundle map[string]interface{}
	if err := json.Unmarshal(respBody, &bundle); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Bundle created successfully:\n")
	fmt.Printf("  ID: %s\n", bundle["bundle_id"])
	fmt.Printf("  Name: %s\n", bundle["name"])
	fmt.Printf("  Hash: %s\n", bundle["hash"])
	fmt.Printf("  Key ID: %s\n", bundle["kid"])

	return nil
}

func listBundles(c *cli.Context) error {
	registryURL := c.String("registry-url")
	limit := c.Int("limit")
	offset := c.Int("offset")

	url := fmt.Sprintf("%s/bundles?limit=%d&offset=%d", registryURL, limit, offset)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to list bundles: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to list bundles (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	bundles := result["bundles"].([]interface{})
	fmt.Printf("Bundles (%d total):\n", len(bundles))
	fmt.Println("=====================================")

	for _, bundle := range bundles {
		bund := bundle.(map[string]interface{})
		fmt.Printf("ID: %s\n", bund["bundle_id"])
		fmt.Printf("  Name: %s\n", bund["name"])
		fmt.Printf("  Hash: %s\n", bund["hash"])
		fmt.Printf("  Key ID: %s\n", bund["kid"])
		fmt.Printf("  Created By: %s\n", bund["created_by"])
		fmt.Printf("  Created: %s\n", bund["created_at"])
		fmt.Println()
	}

	return nil
}

// Health check functions

func checkHealth(c *cli.Context) error {
	serviceURL := c.String("service-url")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(serviceURL + "/healthz")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var health map[string]interface{}
	if err := json.Unmarshal(respBody, &health); err != nil {
		return fmt.Errorf("failed to parse health response: %w", err)
	}

	fmt.Printf("Health Status: %s\n", health["status"])
	fmt.Printf("Timestamp: %s\n", health["timestamp"])
	fmt.Printf("Version: %s\n", health["version"])

	if services, ok := health["services"].(map[string]interface{}); ok {
		fmt.Println("Services:")
		for service, status := range services {
			fmt.Printf("  %s: %s\n", service, status)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service is not healthy (status %d)", resp.StatusCode)
	}

	return nil
}





