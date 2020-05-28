package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	//"net/http"
)

// GiphySearchResponse - What comes back from Giphy
type GiphySearchResponse struct {
	Data       []GifObject `json:"data"`
	Pagination Pagination  `json:"pagination"`
	Meta       Meta        `json:"meta"`
}

// GifObject - the core of the response from Giphy - get your URLs here
type GifObject struct {
	Type             string `json:"type"`
	ID               string `json:"id"`
	URL              string `json:"url"`
	Slug             string `json:"slug"`
	BitlyGifURL      string `json:"bitly_gif_url"`
	BitlyURL         string `json:"bitly_url"`
	EmbedURL         string `json:"embed_url"`
	Username         string `json:"username"`
	Source           string `json:"source"`
	Title            string `json:"title"`
	Rating           string `json:"rating"`
	ContentURL       string `json:"content_url"`
	SourceTld        string `json:"source_tld"`
	SourcePostURL    string `json:"source_post_url"`
	IsSticker        int    `json:"is_sticker"`
	ImportDatetime   string `json:"import_datetime"`
	TrendingDatetime string `json:"trending_datetime"`
}

// Pagination - Page counters for paginated results
type Pagination struct {
	TotalCount int `json:"total_count"`
	Count      int `json:"count"`
	Offset     int `json:"offset"`
}

// Meta - Metadata attacted by Giphy to the result
type Meta struct {
	Status     int    `json:"status"`
	Msg        string `json:"msg"`
	ResponseID string `json:"response_id"`
}

const giphyendpoint = "https://api.giphy.com/v1/gifs/search"
const giphyhost = "api.giphy.com"

var defaultapikey = ""
var passthruapikey = false
var port int

// SearchRequest - a minimal Request to search
type SearchRequest struct {
	Query  string `form:"q"`
	APIKey string `form:"api_key"`
	Limit  int    `form:"limit"`
}

func main() {
	r := gin.New()

	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
		return fmt.Sprintf("%s - [%s] \"%d %s %s\"\n",
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.StatusCode,
			param.Latency,
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	var err error
	var ok bool

	defaultapikey, ok = os.LookupEnv("GIPHYAPIKEY")

	if !ok {
		log.Println("No GIPHYAPIKEY in environment")
		os.Exit(1)
	}

	tmppassthrough, ok := os.LookupEnv("GIPHYKEYPASSTHROUGH")
	if !ok {
		passthruapikey = false
	} else {
		passthruapikey, err = strconv.ParseBool(tmppassthrough)
		if err != nil {
			log.Println("GIPHYKEYPASSTHROUGH not a boolean value - setting off")
			passthruapikey = false
		}
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	r.GET("/v1/gifs/search", processSearch)

	r.NoRoute(proxyAll)

	r.Run(":" + port)
}

func processSearch(c *gin.Context) {
	var searchrequest SearchRequest

	if c.ShouldBind(&searchrequest) == nil {
		newparams := url.Values{}
		newparams.Add("q", searchrequest.Query)
		if searchrequest.APIKey == "" {
			newparams.Add("api_key", defaultapikey)

		} else {
			newparams.Add("api_key", searchrequest.APIKey)
		}

		if searchrequest.Limit == 0 {
			newparams.Add("limit", "10")
		} else {
			newparams.Add("limit", strconv.Itoa(searchrequest.Limit))
		}

		baseURL, err := url.Parse(giphyendpoint)
		if err != nil {
			c.Status(500)
			return
		}

		baseURL.RawQuery = newparams.Encode()
		res, err := http.Get(baseURL.String())
		if err != nil {
			c.Status(404)
			return
		}
		defer res.Body.Close()

		w := c.Writer

		for headername, values := range res.Header {
			w.Header()[headername] = values
		}

		w.WriteHeader(res.StatusCode)

		var buf bytes.Buffer

		io.Copy(w, io.TeeReader(res.Body, &buf))

		var giphysearchresponse GiphySearchResponse

		err = json.NewDecoder(&buf).Decode(&giphysearchresponse)

		// Example - Dump the entire response
		// fmt.Printf("%#v\n", giphysearchresponse)

		// Example - iterate over the decoded response
		//
		// for _, v := range giphysearchresponse.Data {
		// 	fmt.Println(v.Type, v.EmbedURL)
		// }

		return
	}
}

func proxyAll(c *gin.Context) {
	newURL := c.Request.URL

	newURL.Scheme = "https"
	newURL.Host = giphyhost

	res, err := http.Get(newURL.String())
	if err != nil {
		c.Status(404)
		return
	}
	defer res.Body.Close()

	w := c.Writer

	for headername, values := range res.Header {
		w.Header()[headername] = values
	}

	w.WriteHeader(res.StatusCode)

	io.Copy(w, res.Body)

	return

}
