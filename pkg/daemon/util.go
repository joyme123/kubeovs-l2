package daemon

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
)

type ResponseData map[string]interface{}

func NewResponseData() map[string]interface{} {
	return make(map[string]interface{}, 0)
}

// JSONResponse json格式的响应
type JSONResponse struct {
	Code interface{} `json:"code"`
	Data interface{} `json:"data"`
}

// JSONErrorResponse json格式的错误响应
type JSONErrorResponse struct {
	Code    interface{} `json:"code"`
	Details string      `json:"details"`
}

func writeOkResponse(w http.ResponseWriter, m interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(m); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func writeErrorResponse(w http.ResponseWriter, errorCode int, errorMsg string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(errorCode)
	json.NewEncoder(w).Encode(&JSONErrorResponse{Code: 1, Details: errorMsg})
}

// parseResolvConf parses an existing resolv.conf in to a DNS struct
func parseResolvConf(filename string) (*types.DNS, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	dns := types.DNS{}
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip comments, empty lines
		if len(line) == 0 || line[0] == '#' || line[0] == ';' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "nameserver":
			dns.Nameservers = append(dns.Nameservers, fields[1])
		case "domain":
			dns.Domain = fields[1]
		case "search":
			dns.Search = append(dns.Search, fields[1:]...)
		case "options":
			dns.Options = append(dns.Options, fields[1:]...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &dns, nil
}

func GenerateSocketPath() string {
	return DefaultKubeOVSDirectory + "kubeovs.sock"
}
