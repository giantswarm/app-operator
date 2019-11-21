package helmclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
)

// PullChartTarball downloads a tarball from the provided tarball URL,
// returning the file path.
func (c *Client) PullChartTarball(ctx context.Context, tarballURL string) (string, error) {
	c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient tarballURL %#q", tarballURL))

	req, err := c.newRequest("GET", tarballURL)
	if err != nil {
		return "", microerror.Mask(err)
	}

	c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient request URL %#q", req.URL))
	c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient request scheme %#q", req.URL.Scheme))
	c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient request host %#q", req.URL.Host))

	chartTarballPath, err := c.doFile(ctx, req)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return chartTarballPath, nil
}

func (c *Client) doFile(ctx context.Context, req *http.Request) (string, error) {
	var tmpFileName string

	c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient BEFORE request %#q", req.URL))

	req = req.WithContext(ctx)

	c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient AFTER request %#q", req.URL))

	req.URL.Scheme = "https"

	o := func() error {
		resp, err := c.httpClient.Do(req)
		if isNoSuchHostError(err) {
			return backoff.Permanent(microerror.Maskf(executionFailedError, "no such host %#q", req.Host))
		} else if err != nil {
			if resp != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient resp code %d", resp.StatusCode))
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient resp status %#q", resp.Status))

				buf := new(bytes.Buffer)
				_, err = buf.ReadFrom(resp.Body)
				if err != nil {
					return microerror.Mask(err)
				}

				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("DEBUG helmclient StatusCode %d for url %#q with body %s", resp.StatusCode, req.URL.String(), buf.String()))
			} else {
				c.logger.LogCtx(ctx, "level", "info", "message", "DEBUG helmclient nil response")
			}

			return microerror.Mask(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(resp.Body)
			if err != nil {
				return microerror.Mask(err)
			}
			// Github pages 404 produces full HTML page which
			// obscures the logs.
			if resp.StatusCode == http.StatusNotFound {
				return backoff.Permanent(microerror.Maskf(executionFailedError, fmt.Sprintf("got StatusCode %d for url %#q", resp.StatusCode, req.URL.String())))
			}
			return microerror.Maskf(executionFailedError, fmt.Sprintf("got StatusCode %d for url %#q with body %s", resp.StatusCode, req.URL.String(), buf.String()))
		}

		tmpfile, err := afero.TempFile(c.fs, "", "chart-tarball")
		if err != nil {
			return microerror.Mask(err)
		}
		defer tmpfile.Close()

		_, err = io.Copy(tmpfile, resp.Body)
		if err != nil {
			return microerror.Mask(err)
		}

		tmpFileName = tmpfile.Name()

		return nil
	}

	b := backoff.NewMaxRetries(3, 5*time.Second)
	n := backoff.NewNotifier(c.logger, ctx)

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return tmpFileName, nil
}

func (c *Client) newRequest(method, url string) (*http.Request, error) {
	var buf io.Reader

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	req.Header.Set("Accept", "application/json")

	return req, nil
}
