package apijquery

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//Parse the api.jquery.com directory for all *.xml files
//
// return the current api.Api instance
func Parse(directory string) (api *Api, err error) {

	p := &parser{api: NewApi()}
	err = filepath.Walk(directory, p.walk)
	return p.api, err
}

type parser struct {
	api *Api
}

func (p *parser) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	//path is the actual file
	if strings.HasSuffix(path, ".xml") {

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		//read content then try to parse
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		//the content can be either an <entry> or and <entries>
		//
		//Attempt parsing "both" ways and keep only if not empty
		//log.Printf("Parsing %v", path)

		//try as an Entry
		e := new(Entry)
		err = xml.Unmarshal(content, e)
		if err != nil {
			return err
		}

		if e.RawName != "" { //this was an entry (it's not empty)
			p.api.Entry = append(p.api.Entry, e)
		} else { //this was not an "<entry>" (it's empty)
			//try as an "<entries>"
			entries := new(Entries)
			err = xml.Unmarshal(content, entries)
			if err != nil {
				return err
			}

			if len(entries.Entry) != 0 { //there was entries, add them
				p.api.Entries = append(p.api.Entries, entries)
			}
		}
	}
	return nil
}
