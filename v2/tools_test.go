package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	s := testTool.RandomString(10)

	if len(s) != 10 {
		t.Errorf("wrong length random String Returrend, expected 10, but got %d", len(s))
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: true, errorExpected: true},
}

func TestTools_UploadedFiles(t *testing.T) {
	for _, e := range uploadTests {
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes
		uploadFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()

	}

}

func TestTools_UploadOneFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	uploadFiles, err := testTool.UploadOneFile(request, "./testdata/uploads/", true)

	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadFiles.NewFileName))

}

func TestTool_CreateDirIfNotExist(t *testing.T) {
	err := testTool.CreateDirIfNotExist("./testdata/myDir")

	if err != nil {
		t.Error(err)
	}

	err = testTool.CreateDirIfNotExist("./testdata/myDir")

	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "Now is the time for all GOOD men!", expected: "now-is-the-time-for-all-good-men", errorExpected: false},
	{name: "korean string", s: "우 투 더 우투더문", expected: "", errorExpected: true},
	{name: "korean string and roman characters", s: "우 투 더 우투더문 hello world", expected: "hello-world", errorExpected: false},
}

func TestTool_Slugify(t *testing.T) {
	for _, e := range slugTests {
		slug, err := testTool.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected:%s", e.name, err.Error())
		}

		if !e.errorExpected && !strings.EqualFold(e.expected, slug) {
			t.Errorf("%s: wrong slug is returned; expected %s; but got %s;", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	testTool.DownloadStaticFile(rr, req, "./testdata/img.png", "puppy.png")
	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "534283" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.png\"" {
		t.Error("wrong content disposition")
	}

	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name               string
	json               string
	errorExpected      bool
	maxSize            int
	allowUnknownFields bool
}{
	{name: "good json", json: `{"foo":"bar"}`, errorExpected: false, maxSize: 1024, allowUnknownFields: false},
	{name: "badly formatted json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknownFields: false},
	{name: "incorrect type", json: `{"foo":1}`, errorExpected: true, maxSize: 1024, allowUnknownFields: false},
	{name: "two json files", json: `{"foo":"1"}{"alpha":"beta"}`, errorExpected: true, maxSize: 1024, allowUnknownFields: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknownFields: false},
	{name: "syntax error in json", json: `{"foo":1"`, errorExpected: true, maxSize: 1024, allowUnknownFields: false},
	{name: "unknown field  in json", json: `{"fooo":"1"}`, errorExpected: true, maxSize: 1024, allowUnknownFields: false},
	{name: "allow unknown field  in json", json: `{"fooo":"1"}`, errorExpected: false, maxSize: 1024, allowUnknownFields: true},
	{name: "missing field in json", json: `{jack:"1"}`, errorExpected: true, maxSize: 1024, allowUnknownFields: true},
	{name: "file too large", json: `{"foo":"1"}`, errorExpected: true, maxSize: 5, allowUnknownFields: true},
	{name: "not json", json: `Hello World`, errorExpected: true, maxSize: 1024, allowUnknownFields: true},
}

func TestTools_ReadJSON(t *testing.T) {
	for _, e := range jsonTests {
		testTool.MaxJSONSize = e.maxSize

		testTool.AllowUnknownFields = e.allowUnknownFields

		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none recieved", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one recieved: %s", e.name, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTool.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("Failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	err := testTool.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding JSON", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}

}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})

	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"
	_, _, err := testTool.PushJSONToRemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url:", err)
	}
}
