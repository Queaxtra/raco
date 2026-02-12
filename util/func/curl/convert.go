package curl

import (
	"raco/model"
	"strings"
)

func Convert(req *model.Request) string {
	var builder strings.Builder

	builder.WriteString("curl -X ")
	builder.WriteString(req.Method)
	builder.WriteString(" '")
	builder.WriteString(req.URL)
	builder.WriteString("'")

	for key, value := range req.Headers {
		builder.WriteString(" -H '")
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteString("'")
	}

	hasBody := req.Body != ""
	if hasBody {
		builder.WriteString(" -d '")
		builder.WriteString(req.Body)
		builder.WriteString("'")
	}

	return builder.String()
}
