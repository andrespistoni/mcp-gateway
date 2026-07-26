package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const ProtocolVersion = "2024-11-05"

type EnvelopeKind uint8

const (
	Request EnvelopeKind = iota + 1
	Notification
	Result
	Error
)

type RawID struct {
	raw json.RawMessage
}

func ParseID(raw json.RawMessage) (RawID, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return RawID{}, fmt.Errorf("falta id JSON-RPC")
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return RawID{}, fmt.Errorf("id JSON-RPC inválido")
	}
	switch value.(type) {
	case string, json.Number:
		return RawID{raw: append(json.RawMessage(nil), trimmed...)}, nil
	default:
		return RawID{}, fmt.Errorf("id JSON-RPC debe ser string o número")
	}
}

func StringID(value string) RawID {
	raw, _ := json.Marshal(value)
	return RawID{raw: raw}
}

func NumberID(value int64) RawID {
	return RawID{raw: json.RawMessage(fmt.Sprintf("%d", value))}
}

func (id RawID) Bytes() json.RawMessage {
	return append(json.RawMessage(nil), id.raw...)
}

func (id RawID) Equal(other RawID) bool {
	return bytes.Equal(id.raw, other.raw)
}

func (id RawID) MarshalJSON() ([]byte, error) {
	if _, err := ParseID(id.raw); err != nil {
		return nil, err
	}
	return append([]byte(nil), id.raw...), nil
}

type RPCError struct {
	Code    int64
	Message string
	Data    json.RawMessage
	fields  map[string]json.RawMessage
}

func (e RPCError) Fields() map[string]json.RawMessage {
	return cloneFields(e.fields)
}

type Envelope struct {
	kind   EnvelopeKind
	id     RawID
	hasID  bool
	method string
	params json.RawMessage
	result json.RawMessage
	rpcErr *RPCError
	fields map[string]json.RawMessage
}

func ParseEnvelope(data []byte) (Envelope, error) {
	if !utf8.Valid(data) {
		return Envelope{}, fmt.Errorf("JSON-RPC no es UTF-8 válido")
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return Envelope{}, fmt.Errorf("JSON-RPC debe ser un objeto único")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return Envelope{}, fmt.Errorf("objeto JSON-RPC inválido")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Envelope{}, fmt.Errorf("JSON-RPC contiene valores adicionales")
	}
	var version string
	if err := json.Unmarshal(fields["jsonrpc"], &version); err != nil || version != "2.0" {
		return Envelope{}, fmt.Errorf("jsonrpc debe ser 2.0")
	}
	envelope := Envelope{fields: cloneFields(fields)}
	methodRaw, hasMethod := fields["method"]
	if hasMethod {
		if err := json.Unmarshal(methodRaw, &envelope.method); err != nil || envelope.method == "" {
			return Envelope{}, fmt.Errorf("method JSON-RPC inválido")
		}
		if params, ok := fields["params"]; ok {
			envelope.params = append(json.RawMessage(nil), params...)
		}
		if rawID, ok := fields["id"]; ok {
			id, err := ParseID(rawID)
			if err != nil {
				return Envelope{}, err
			}
			envelope.id, envelope.hasID, envelope.kind = id, true, Request
		} else {
			envelope.kind = Notification
		}
		if _, ok := fields["result"]; ok {
			return Envelope{}, fmt.Errorf("request JSON-RPC contiene result")
		}
		if _, ok := fields["error"]; ok {
			return Envelope{}, fmt.Errorf("request JSON-RPC contiene error")
		}
		return envelope, nil
	}

	rawID, hasID := fields["id"]
	if !hasID {
		return Envelope{}, fmt.Errorf("respuesta JSON-RPC sin id")
	}
	id, err := ParseID(rawID)
	if err != nil {
		return Envelope{}, err
	}
	envelope.id, envelope.hasID = id, true
	resultRaw, hasResult := fields["result"]
	errorRaw, hasError := fields["error"]
	if hasResult == hasError {
		return Envelope{}, fmt.Errorf("respuesta JSON-RPC debe contener result o error")
	}
	if hasResult {
		envelope.kind = Result
		envelope.result = append(json.RawMessage(nil), resultRaw...)
		return envelope, nil
	}
	parsedError, err := parseRPCError(errorRaw)
	if err != nil {
		return Envelope{}, err
	}
	envelope.kind = Error
	envelope.rpcErr = &parsedError
	return envelope, nil
}

func parseRPCError(raw json.RawMessage) (RPCError, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return RPCError{}, fmt.Errorf("error JSON-RPC inválido")
	}
	var code json.Number
	if err := json.Unmarshal(fields["code"], &code); err != nil {
		return RPCError{}, fmt.Errorf("código de error JSON-RPC inválido")
	}
	codeValue, err := code.Int64()
	if err != nil {
		return RPCError{}, fmt.Errorf("código de error JSON-RPC inválido")
	}
	var message string
	if err := json.Unmarshal(fields["message"], &message); err != nil {
		return RPCError{}, fmt.Errorf("mensaje de error JSON-RPC inválido")
	}
	parsed := RPCError{Code: codeValue, Message: message, fields: cloneFields(fields)}
	if data, ok := fields["data"]; ok {
		parsed.Data = append(json.RawMessage(nil), data...)
	}
	return parsed, nil
}

func NewRequest(id RawID, method string, params any) (Envelope, error) {
	return newOutbound(&id, method, params)
}

func NewNotification(method string, params any) (Envelope, error) {
	return newOutbound(nil, method, params)
}

func NewResult(id RawID, result any) (Envelope, error) {
	resultRaw, err := json.Marshal(result)
	if err != nil {
		return Envelope{}, err
	}
	fields := map[string]json.RawMessage{
		"jsonrpc": json.RawMessage(`"2.0"`),
		"id":      id.Bytes(),
		"result":  resultRaw,
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return Envelope{}, err
	}
	return ParseEnvelope(encoded)
}

func NewError(id RawID, rpcError RPCError) (Envelope, error) {
	errorFields := cloneFields(rpcError.fields)
	if errorFields == nil {
		errorFields = make(map[string]json.RawMessage)
	}
	codeRaw, _ := json.Marshal(rpcError.Code)
	messageRaw, _ := json.Marshal(rpcError.Message)
	errorFields["code"] = codeRaw
	errorFields["message"] = messageRaw
	if rpcError.Data != nil {
		errorFields["data"] = append(json.RawMessage(nil), rpcError.Data...)
	}
	errorRaw, err := json.Marshal(errorFields)
	if err != nil {
		return Envelope{}, err
	}
	fields := map[string]json.RawMessage{
		"jsonrpc": json.RawMessage(`"2.0"`),
		"id":      id.Bytes(),
		"error":   errorRaw,
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return Envelope{}, err
	}
	return ParseEnvelope(encoded)
}

func newOutbound(id *RawID, method string, params any) (Envelope, error) {
	if method == "" {
		return Envelope{}, fmt.Errorf("method JSON-RPC vacío")
	}
	fields := map[string]json.RawMessage{"jsonrpc": json.RawMessage(`"2.0"`)}
	methodRaw, _ := json.Marshal(method)
	fields["method"] = methodRaw
	if params != nil {
		paramsRaw, err := json.Marshal(params)
		if err != nil {
			return Envelope{}, err
		}
		fields["params"] = paramsRaw
	}
	if id != nil {
		if _, err := ParseID(id.raw); err != nil {
			return Envelope{}, err
		}
		fields["id"] = id.Bytes()
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return Envelope{}, err
	}
	return ParseEnvelope(encoded)
}

func (e Envelope) Kind() EnvelopeKind { return e.kind }
func (e Envelope) ID() (RawID, bool)  { return e.id, e.hasID }
func (e Envelope) Method() string     { return e.method }
func (e Envelope) Params() json.RawMessage {
	return append(json.RawMessage(nil), e.params...)
}
func (e Envelope) Result() json.RawMessage {
	return append(json.RawMessage(nil), e.result...)
}
func (e Envelope) RPCError() (RPCError, bool) {
	if e.rpcErr == nil {
		return RPCError{}, false
	}
	copyValue := *e.rpcErr
	copyValue.Data = append(json.RawMessage(nil), e.rpcErr.Data...)
	copyValue.fields = cloneFields(e.rpcErr.fields)
	return copyValue, true
}
func (e Envelope) Fields() map[string]json.RawMessage { return cloneFields(e.fields) }

func (e Envelope) MarshalJSON() ([]byte, error) {
	if len(e.fields) == 0 {
		return nil, fmt.Errorf("envelope JSON-RPC vacío")
	}
	return json.Marshal(e.fields)
}

func cloneFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	clone := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		clone[key] = append(json.RawMessage(nil), value...)
	}
	return clone
}
