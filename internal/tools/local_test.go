package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoogleSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := GoogleSearch(ctx, &SearchRequest{
		Query: "马尔代夫的珊瑚为什么白化了？",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp.Results)
}

func TestFetchUrl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := FetchUrl(ctx, &FetchUrlRequest{
		URL: "https://www.baidu.com",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}
