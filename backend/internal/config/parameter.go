package config

import (
	"context"
	"fmt"

	parametermanager "cloud.google.com/go/parametermanager/apiv1"
	"cloud.google.com/go/parametermanager/apiv1/parametermanagerpb"
)

// renderLatest returns the newest version of a Parameter Manager parameter, with its
// __REF__(...) secret references replaced by the secrets' values. It authenticates with
// Application Default Credentials: the service account on Cloud Run, or gcloud locally.
// A variable so tests can replace it.
var renderLatest = func(ctx context.Context, parameter string) ([]byte, error) {
	client, err := parametermanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create parameter manager client: %w", err)
	}
	defer client.Close()

	resp, err := client.RenderParameterVersion(ctx, &parametermanagerpb.RenderParameterVersionRequest{
		Name: parameter + "/versions/latest",
	})
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", parameter, err)
	}
	return resp.RenderedPayload, nil
}
