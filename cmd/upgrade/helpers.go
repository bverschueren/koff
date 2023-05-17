/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/gmeghnag/koff/vars"
	progressbar "github.com/schollz/progressbar/v3"
)

type Releases []Release
type Release map[string]interface{}

func updateKoffExecutable(koffExecutablePath string, url string, desiredVersion string) (err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error: Expected response code 200 from request executed to " + url + ", received " + strconv.Itoa(resp.StatusCode))
	}
	defer resp.Body.Close()

	err = os.Remove(koffExecutablePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.OpenFile(koffExecutablePath, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	//bar := progressbar.Default(-1, "")
	bar := CustomBytes(desiredVersion,
		resp.ContentLength,
		"upgrading",
	)

	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func checkReleases(repoName string) {
	resp, err := http.Get("https://api.github.com/repos/" + repoName + "/releases")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		panic(err)
	}
	var koffReleases Releases
	err = json.Unmarshal(body, &koffReleases)
	if err != nil {
		panic(err)
	}

	if vars.KoffTag == "" {
		vars.KoffTag = "v0.9.1"
	}
	fmt.Println("koff version is " + vars.KoffTag)
	fmt.Println("")
	fmt.Println("Available updates:")
	fmt.Println("")
	currentVer := semver.New(vars.KoffTag[1:])
	for _, release := range koffReleases {
		availableRelease := release["tag_name"].(string)
		availableReleaseVer := semver.New(availableRelease[1:])
		if currentVer.LessThan(*availableReleaseVer) {
			fmt.Println(availableRelease)
		}
	}
}

func CustomBytes(desiredVersion string, maxBytes int64, description ...string) *progressbar.ProgressBar {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	return progressbar.NewOptions64(
		maxBytes,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(35),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\rkoff upgraded to "+desiredVersion+"                                                                        \n")
		}),
		progressbar.OptionSpinnerType(14),
		//progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}
