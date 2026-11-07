package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

// FirestoreClient wraps the Google Cloud Firestore client.
type FirestoreClient struct {
	Client             *firestore.Client
	ProjectID          string
	DatabaseID         string
	ProjectsCollection string
}

// NewFirestoreClient initializes a live connection to Google Cloud Firestore for the given GCP project ID,
// database ID, and root collection name (e.g. "jp-dev-projects", "jp-stg-projects", "jp-prd-projects").
// It instruments the underlying gRPC client with OpenTelemetry so that database operations generate child spans under active request traces.
func NewFirestoreClient(ctx context.Context, gcpProjectID, databaseID, projectsCollection string) (*FirestoreClient, error) {
	if gcpProjectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT is required to initialize Firestore client")
	}

	dbID := "(default)"
	if databaseID != "" {
		dbID = databaseID
	}

	colName := projectsCollection
	if colName == "" {
		colName = "jp-dev-projects"
	}

	// Instrument gRPC connection with OpenTelemetry stats handler
	grpcOpt := option.WithGRPCDialOption(grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	var (
		client *firestore.Client
		err    error
	)
	if dbID == "" || dbID == "(default)" {
		client, err = firestore.NewClient(ctx, gcpProjectID, grpcOpt)
	} else {
		client, err = firestore.NewClientWithDatabase(ctx, gcpProjectID, dbID, grpcOpt)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}

	return &FirestoreClient{
		Client:             client,
		ProjectID:          gcpProjectID,
		DatabaseID:         dbID,
		ProjectsCollection: colName,
	}, nil
}

// Close closes the underlying Firestore client connection.
func (fc *FirestoreClient) Close() error {
	if fc.Client != nil {
		return fc.Client.Close()
	}
	return nil
}

// Projects returns the top-level projects collection reference (e.g. "jp-dev-projects").
func (fc *FirestoreClient) Projects() *firestore.CollectionRef {
	collectionName := fc.ProjectsCollection
	if collectionName == "" {
		collectionName = "jp-dev-projects"
	}
	return fc.Client.Collection(collectionName)
}

// ProjectDoc returns the document reference for a specific project.
func (fc *FirestoreClient) ProjectDoc(projectID string) *firestore.DocumentRef {
	return fc.Projects().Doc(projectID)
}
