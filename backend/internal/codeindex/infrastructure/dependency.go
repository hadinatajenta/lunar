package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

var ipv4Pattern = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

func httpClientDependency(repoName string, sourceFile string, endpoint string) (codeindexdomain.ServiceDependency, bool) {
	targetService := extractServiceFromURL(endpoint, repoName)
	if targetService == "" {
		return codeindexdomain.ServiceDependency{}, false
	}
	return codeindexdomain.ServiceDependency{
		FromService: repoName,
		ToService:   targetService,
		CallType:    string(codeindexdomain.CallTypeHTTPClient),
		Endpoint:    endpoint,
		SourceFile:  sourceFile,
	}, true
}

func kafkaDependency(repoName string, sourceFile string, topic string, callType codeindexdomain.CallType) codeindexdomain.ServiceDependency {
	return codeindexdomain.ServiceDependency{
		FromService: repoName,
		CallType:    string(callType),
		Topic:       topic,
		SourceFile:  sourceFile,
	}
}

func extractServiceFromURL(rawURL string, selfService string) string {
	host := strings.TrimPrefix(rawURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	if index := strings.IndexAny(host, "/:"); index > 0 {
		host = host[:index]
	}

	if host == "" || strings.Contains(host, selfService) {
		return ""
	}
	if isIPAddress(host) || strings.ContainsAny(host, "{}$") {
		return ""
	}
	if strings.Contains(strings.ToLower(host), "env") {
		return ""
	}
	return host
}

func isIPAddress(value string) bool {
	return ipv4Pattern.MatchString(value) || value == "localhost" || value == "127.0.0.1"
}

func (s *Store) GetServiceDependencies(ctx context.Context, serviceName string) ([]codeindexdomain.ServiceDependency, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT from_service, to_service, call_type, endpoint, topic, source_file
		FROM service_deps
		WHERE from_service = ? OR to_service = ?
		ORDER BY call_type, from_service
	`, serviceName, serviceName)
	if err != nil {
		return nil, fmt.Errorf("getServiceDependencies: %w", err)
	}
	defer rows.Close()

	var dependencies []codeindexdomain.ServiceDependency
	for rows.Next() {
		var dependency codeindexdomain.ServiceDependency
		if err := rows.Scan(
			&dependency.FromService,
			&dependency.ToService,
			&dependency.CallType,
			&dependency.Endpoint,
			&dependency.Topic,
			&dependency.SourceFile,
		); err != nil {
			return nil, fmt.Errorf("getServiceDependencies: scan: %w", err)
		}
		dependencies = append(dependencies, dependency)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getServiceDependencies: %w", err)
	}
	return dependencies, nil
}

func (s *Store) SaveDependencies(ctx context.Context, repoName string, dependencies []codeindexdomain.ServiceDependency) error {
	if len(dependencies) == 0 {
		return nil
	}

	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("saveDependencies: begin transaction: %w", err)
	}
	defer rollbackQuietly(transaction)

	if err := insertDependencies(ctx, transaction, repoName, dependencies); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("saveDependencies: commit: %w", err)
	}
	return nil
}

func insertDependencies(ctx context.Context, transaction *sql.Tx, repoName string, dependencies []codeindexdomain.ServiceDependency) error {
	if len(dependencies) == 0 {
		return nil
	}

	statement, err := transaction.PrepareContext(ctx, `
		INSERT OR IGNORE INTO service_deps(from_service, to_service, call_type, endpoint, topic, source_file)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("insertDependencies: prepare: %w", err)
	}
	defer func() { _ = statement.Close() }()

	for _, dependency := range dependencies {
		fromService := dependency.FromService
		if fromService == "" {
			fromService = repoName
		}
		if _, err := statement.ExecContext(
			ctx,
			fromService,
			dependency.ToService,
			dependency.CallType,
			dependency.Endpoint,
			dependency.Topic,
			dependency.SourceFile,
		); err != nil {
			return fmt.Errorf("insertDependencies: insert %s: %w", dependency.SourceFile, err)
		}
	}
	return nil
}

func deleteFileDependencies(ctx context.Context, transaction *sql.Tx, repoName string, filePath string) error {
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM service_deps WHERE from_service = ? AND source_file = ?",
		repoName,
		filePath,
	); err != nil {
		return fmt.Errorf("deleteFileDependencies: %w", err)
	}
	return nil
}
