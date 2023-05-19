package respond

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	EncodeJSON = 0
	EncodeXML = 1
)

// A problem details object conforming to RFC 7807 §3.1.
// https://datatracker.ietf.org/doc/html/rfc7807#section-3.1
// 
// TODO: Examples.
type Problem struct {
	Type string
	Title string
	Status int
	Detail string
	Instance string
	// Non-standard fields. Values are serialized using json.Marshal()
	Extensions map[string]any
}

func (p Problem) MarshalJSON() ([]byte, error) {
	buf := []byte{ '{' }
	if p.Type != "" {
		buf = append(buf, fmt.Sprintf(`"type":"%s",`, p.Type)...)
	}
	if p.Title != "" {
		buf = append(buf, fmt.Sprintf(`"title":"%s",`, p.Title)...)
	}
	if p.Status != 0 {
		buf = append(buf, fmt.Sprintf(`"status":%d,`, p.Status)...)
	}
	if p.Detail != "" {
		buf = append(buf, fmt.Sprintf(`"detail":"%s",`, p.Detail)...)
	}
	if p.Instance != "" {
		buf = append(buf, fmt.Sprintf(`"instance":"%s",`, p.Instance)...)
	}
	for key, value := range p.Extensions {
		value, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		buf = append(buf, fmt.Sprintf(`"%s":%s,`, key, value)...)
	}
	if buf[len(buf)-1] == ',' {
		buf[len(buf)-1] = '}'
	} else {
		buf = append(buf, '}')
	}
	return buf, nil
}

func WithProblem(w http.ResponseWriter, enc int, p Problem) error {
	var bs []byte
	var err error
	switch enc {
	case EncodeJSON:
		w.Header().Set("Content-Type", "application/problem+json")
		bs, err = json.Marshal(p)
		if err != nil {
			return fmt.Errorf("failed to encode Problem as JSON: %v", err)
		}
	case EncodeXML:
		w.Header().Set("Content-Type", "application/problem+xml")
		panic("TODO")
	default:
		return fmt.Errorf("unknown encoding supplied, use EncodeJSON or EncodeXML")
	}
	
	if p.Status != 0 {
		w.WriteHeader(p.Status)
	} else {
		w.WriteHeader(400)
	}
	_, err = w.Write(bs)
	if err != nil {
		return err
	}

	return nil
}