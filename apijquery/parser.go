package apijquery

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//Parse the api.jquery.com directory for all *.xml files
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
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		defer file.Close()

		//Attempt parsing "both" ways
		//log.Printf("Parsing %v", path)
		err = p.parseEntry(content)
		if err != nil {
			log.Printf("Not an entry %v", path)
			return err
		}
		err = p.parseEntries(content, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) parseEntry(content []byte) (err error) {
	//try to read an "entry"
	e := new(Entry)
	err = xml.Unmarshal(content, e)
	if err != nil {
		return
	}

	if e.RawName == "" {
		return nil
	}
	//log.Printf("%v signatures %v", e.RawName, len(e.Signature))

	p.api.Entry = append(p.api.Entry, e)
	return
}

func (p *parser) parseEntries(content []byte, path string) (err error) {
	//try to read an "entry"
	entries := new(Entries)
	err = xml.Unmarshal(content, entries)
	if err != nil {
		log.Printf("not an entries")
		return
	}

	if len(entries.Entry) == 0 {
		//log.Printf("empty entries")
		return nil
	}

	//log.Printf("adding  entries %v %v", entries.Desc, len(entries.Entry))
	p.api.Entries = append(p.api.Entries, entries)
	return nil
}
