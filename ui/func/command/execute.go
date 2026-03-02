package command

import (
	"raco/http"
	"raco/model"

	tea "github.com/charmbracelet/bubbletea"
)

func copyAndReplaceHeaders(headers map[string]string, env *model.Environment) map[string]string {
	if headers == nil {
		return nil
	}
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		out[k] = http.ReplaceEnvVars(v, env)
	}
	return out
}

type RequestExecutedMsg struct {
	Response         *model.Response
	Error            string
	AssertionResults []model.AssertionResult
}

func Execute(client *http.Client, req *model.Request, env *model.Environment) tea.Cmd {
	return func() tea.Msg {
		if req == nil {
			return RequestExecutedMsg{Response: nil}
		}

		processedReq := *req
		processedReq.URL = http.ReplaceEnvVars(req.URL, env)
		processedReq.Body = http.ReplaceEnvVars(req.Body, env)
		processedReq.Headers = copyAndReplaceHeaders(req.Headers, env)
		if len(req.Query) > 0 {
			processedReq.Query = http.ReplaceEnvVarsInMap(req.Query, env)
		}
		processedReq.Assertions = req.Assertions
		processedReq.Extractors = req.Extractors

		resp, err := client.Execute(&processedReq)
		if err != nil {
			return RequestExecutedMsg{Response: nil, Error: err.Error()}
		}

		results := make([]model.AssertionResult, 0, len(req.Assertions))
		for _, assertion := range req.Assertions {
			result := model.ValidateAssertion(assertion, resp)
			results = append(results, result)
		}

		if len(req.Extractors) > 0 {
			if env != nil {
				for _, extractor := range req.Extractors {
					_ = model.ExtractValue(extractor, resp, env)
				}
			}
		}

		return RequestExecutedMsg{Response: resp, AssertionResults: results}
	}
}
