package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

type Server struct {
	Host    string
	Section string
	Token   string
	Videos  []Video
}

var serverRegex = regexp.MustCompile(`^((.+)@)?(([^:]+)(:\d+)?)/(\d+)$`)

func ServerFromArg(arg string) (*Server, error) {
	var s Server
	r := serverRegex.FindStringSubmatch(arg)
	if len(r) == 0 {
		return nil, fmt.Errorf("Invalid server specified")
	}

	token := r[2]
	host := r[4]
	port := r[5]
	section := r[6]
	if port == "" {
		port = ":32400"
	}
	if token == "" {
		token = os.Getenv("PLEX_TOKEN")
	}

	s.Token = token
	s.Host = fmt.Sprintf("%s%s", host, port)
	s.Section = section

	return &s, nil
}

func (s *Server) FetchSection() error {
	url := fmt.Sprintf("http://%s/library/sections/%s/allLeaves?X-Plex-Token=%s", s.Host, s.Section, s.Token)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var r SectionResponse

	err = xml.Unmarshal(body, &r)
	if err != nil {
		return err
	}

	s.Videos = r.Videos

	return nil
}

func (s *Server) PopulateGUID(v *Video) error {
	url := fmt.Sprintf("http://%s%s?X-Plex-Token=%s", s.Host, v.Key, s.Token)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var r SectionResponse

	err = xml.Unmarshal(body, &r)
	if err != nil {
		return err
	}

	if len(r.Videos) == 0 {
		return fmt.Errorf("Not found")
	}

	v.OfficialGUID = r.Videos[0].OfficialGUID
	return nil
}

func (s *Server) MarkWatched(v *Video) error {
	url := fmt.Sprintf("http://%s/:/scrobble?identifier=com.plexapp.plugins.library&key=%s&X-Plex-Token=%s", s.Host, v.ID(), s.Token)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Server) MarkUnwatched(v *Video) error {
	url := fmt.Sprintf("http://%s/:/unscrobble?identifier=com.plexapp.plugins.library&key=%s&X-Plex-Token=%s", s.Host, v.ID(), s.Token)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type Video struct {
	GrandparentTitle string `xml:"grandparentTitle,attr"`
	Index            string `xml:"index,attr"`
	Key              string `xml:"key,attr"`
	OfficialGUID     string `xml:"guid,attr"`
	ParentIndex      string `xml:"parentIndex,attr"`
	Title            string `xml:"title,attr"`
	ViewCount        string `xml:"viewCount,attr"`
	Year             string `xml:"year,attr"`
}

var idRegex = regexp.MustCompile(`/library/metadata/(\d+)`)

func (v *Video) ID() string {
	r := idRegex.FindStringSubmatch(v.Key)
	if len(r) > 1 {
		return r[1]
	}
	return ""
}

func (v *Video) GUID(fetch bool) string {
	if v.OfficialGUID != "" {
		return v.OfficialGUID
	}

	return fmt.Sprintf("%s - %s - %s - %s - %s", v.GrandparentTitle, v.ParentIndex, v.Index, v.Year, v.Title)
}

func (v *Video) Watched() bool {
	number, err := strconv.ParseInt(v.ViewCount, 10, 0)
	if err != nil {
		return false
	}

	return number > 0
}

type SectionResponse struct {
	Videos []Video `xml:"Video"`
}
