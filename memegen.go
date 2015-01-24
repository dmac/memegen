package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"unicode"
)

var MemesFile = "memes.json"

type Meme struct {
	Id     string
	Name   string
	URL    string
	Width  int
	Height int
}

func DownloadMemesFile() error {
	fmt.Println("Downloading memes file to", MemesFile)

	resp, err := http.Get("https://api.imgflip.com/get_memes")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	memes := data["data"].(map[string]interface{})["memes"]
	memesJson, err := json.Marshal(memes)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(MemesFile, memesJson, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadMemes() ([]Meme, error) {
	if _, err := os.Stat(MemesFile); os.IsNotExist(err) {
		if err := DownloadMemesFile(); err != nil {
			return nil, err
		}
	}

	jsn, err := ioutil.ReadFile(MemesFile)
	if err != nil {
		return nil, err
	}

	var memes []Meme
	err = json.Unmarshal(jsn, &memes)
	if err != nil {
		return nil, err
	}

	return memes, nil
}

func ShortName(name string) string {
	f := func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	}
	return strings.ToLower(strings.Join(strings.FieldsFunc(name, f), ""))
}

func PrintMemes(memes []Meme) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, ' ', 0)
	for _, meme := range memes {
		fmt.Fprintf(w, "%s\t%s\n", ShortName(meme.Name), meme.URL)
	}
	w.Flush()
}

func ChooseMeme(memes []Meme, needle string) (Meme, bool) {
	for _, meme := range memes {
		if strings.Contains(ShortName(meme.Name), needle) {
			return meme, true
		}
	}
	return Meme{}, false
}

func GenerateMeme(meme Meme, topText string, bottomText string) (string, error) {
	username := os.Getenv("IMGFLIP_USERNAME")
	password := os.Getenv("IMGFLIP_PASSWORD")
	if username == "" {
		return "", errors.New("Missing environment variable: IMGFLIP_USERNAME\n" +
			"Sign up at https://imgflip.com/signup")
	}
	if password == "" {
		return "", errors.New("Missing environment variable: IMGFLIP_PASSWORD\n" +
			"Sign up at https://imgflip.com/signup")
	}

	values := url.Values{}
	values.Set("template_id", meme.Id)
	values.Set("username", username)
	values.Set("password", password)
	values.Set("text0", topText)
	values.Set("text1", bottomText)

	resp, err := http.PostForm("https://api.imgflip.com/caption_image", values)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	if !data["success"].(bool) {
		return "", errors.New(data["error_message"].(string))
	}

	url := data["data"].(map[string]interface{})["url"].(string)

	return url, nil
}

func PrintUsage() {
	fmt.Println("Usage: memegen <MEME> ['<TOP TEXT>'] [<'BOTTOM TEXT'>]")
}

func main() {
	memes, err := LoadMemes()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	switch len(os.Args) {
	case 1:
		PrintMemes(memes)
		PrintUsage()
	case 2:
		meme, ok := ChooseMeme(memes, os.Args[1])
		if !ok {
			fmt.Println("Error: No meme found")
			os.Exit(1)
		}
		fmt.Printf("%s\t%s\n", ShortName(meme.Name), meme.URL)
	case 3, 4:
		meme, ok := ChooseMeme(memes, os.Args[1])
		if !ok {
			fmt.Println("Error: No meme found")
			os.Exit(1)
		}
		topText := os.Args[2]
		bottomText := ""
		if len(os.Args) == 4 {
			bottomText = os.Args[3]
		}
		url, err := GenerateMeme(meme, topText, bottomText)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s\t%s\n", ShortName(meme.Name), url)
	default:
		PrintUsage()
	}
}
