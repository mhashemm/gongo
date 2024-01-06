package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	re          = regexp.MustCompile(`(?msU)query\s+(.*)\s*\((.*)\)\s*=\s*([{\[].*[}\]])\s*;`)
	formatValue = regexp.MustCompile(`,([}\]])`)
	whiteSpaces = regexp.MustCompile(`\s+`)
	formatParam = regexp.MustCompile(`,(\w)`)
	param       = regexp.MustCompile(`%(\w)`)
)

func files(parent string, arr []string) []string {
	dir, err := os.ReadDir(parent)
	if err != nil {
		panic(err)
	}

	for _, f := range dir {
		if f.IsDir() {
			files(parent+f.Name()+"/", arr)
		} else if strings.HasSuffix(f.Name(), ".gongo") {
			arr = append(arr, parent+f.Name())
		}
	}
	return arr
}

type Query struct {
	Name   string
	Params string
	Value  any
}

func parse(content string) []Query {
	queries := make([]Query, 0)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		name, params, value := match[1], match[2], match[3]
		var obj any
		err := json.Unmarshal([]byte(value), &obj)
		if err != nil {
			panic(match[0])
		}
		queries = append(queries, Query{
			Name:   name,
			Params: params,
			Value:  obj,
		})
	}
	return queries
}

func generate(query Query) string {
	value := formatValue.ReplaceAllString(generateValue(query.Value), "$1")
	params := generateParams(query.Params)
	return fmt.Sprintf(`
func %s(%s) any {
	return %v
}
	`, query.Name, params, value)
}

func generateValue(v any) string {
	str := strings.Builder{}
	switch v := v.(type) {
	case []any:
		str.WriteString("bson.A{")
		for _, o := range v {
			str.WriteString(fmt.Sprintf("%v,", generateValue(o)))
		}
		str.WriteString("}")
		return str.String()
	case map[string]any:
		str.WriteString("bson.D{")
		for k, v := range v {
			str.WriteString(fmt.Sprintf("{Key: \"%v\", Value: %v},", k, generateValue(v)))
		}
		str.WriteString("}")
		return str.String()
	case string:
		if param.MatchString(v) {
			return v[1:]
		}
		return fmt.Sprintf("\"%s\"", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func generateParams(v string) string {
	return formatParam.ReplaceAllString(strings.TrimSpace(whiteSpaces.ReplaceAllString(v, " ")), ", $1")
}

// func save(){

//}

func main() {
	in := *flag.String("in", "./", "parent dicretory for gongos files")
	if !strings.HasSuffix(in, "/") {
		in += "/"
	}
	// out := *flag.String("out", "./", "output dicretory for go files")

	fs := files(in, make([]string, 0))

	for _, f := range fs {
		content, _ := os.ReadFile(f)
		queries := parse(string(content))
		for _, q := range queries {
			// generate(q)
			fmt.Println(generate(q))
		}
	}

}
