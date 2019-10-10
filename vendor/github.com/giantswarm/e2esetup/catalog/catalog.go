package catalog

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/google/go-github/github"
)

func GetLatestChart(ctx context.Context, catalog, app string) (string, error) {
	client := github.NewClient(nil)

	query := fmt.Sprintf("repo:giantswarm/%s filename:%s", catalog, app)
	searchOption := github.SearchOptions{
		Sort: "indexed",
	}
	result, _, err := client.Search.Code(ctx, query, &searchOption)
	if err != nil {
		return "", microerror.Mask(err)
	}

	path := result.CodeResults[0].GetPath()

	r, _, _, err := client.Repositories.GetContents(ctx, "giantswarm", catalog, path, nil)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return r.GetDownloadURL(), nil
}
