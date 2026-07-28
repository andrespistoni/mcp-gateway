package proxy

import (
	"encoding/json"
	"fmt"

	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/project"
)

func (s *Service) routeCall(raw json.RawMessage, directory project.OptionalDir) (string, Route, json.RawMessage, error) {
	fields, err := mcp.DecodeObject(raw)
	if err != nil {
		return "", Route{}, nil, err
	}
	var name string
	if err := json.Unmarshal(fields["name"], &name); err != nil || name == "" {
		return "", Route{}, nil, fmt.Errorf("tools/call.name inválido")
	}
	route, ok := s.catalog.Route(name)
	if !ok {
		return "", Route{}, nil, fmt.Errorf("tool desconocida")
	}
	encodedName, _ := json.Marshal(route.OriginalName)
	fields["name"] = encodedName
	if route.InjectProject && directory.Present() {
		arguments, exists := fields["arguments"]
		if !exists {
			arguments = json.RawMessage(`{}`)
		}
		if object, decodeErr := mcp.DecodeObject(arguments); decodeErr == nil {
			if _, provided := object[route.ProjectArgument]; !provided {
				encoded, _ := json.Marshal(directory.Path())
				object[route.ProjectArgument] = encoded
				arguments, _ = json.Marshal(object)
			}
			fields["arguments"] = arguments
		}
	}
	forwarded, err := json.Marshal(fields)
	if err != nil {
		return "", Route{}, nil, err
	}
	return name, route, forwarded, nil
}
