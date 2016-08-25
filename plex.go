package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

func loadSection(server Server) (*SectionResponse, error) {
	url := fmt.Sprintf("http://%s/library/sections/%s/allLeaves?X-Plex-Token=%s", server.Host, server.Section, server.Token)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r SectionResponse

	err = xml.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
