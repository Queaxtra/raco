package util

import (
	"raco/model"
	"raco/util/func/curl"
)

func ParseCurl(curlCmd string) (*model.Request, error) {
	return curl.Parse(curlCmd)
}

func ToCurl(req *model.Request) string {
	return curl.Convert(req)
}
