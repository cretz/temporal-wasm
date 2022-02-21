package host

import (
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
)

type failure struct {
	Message      string     `json:"message,omitempty"`
	Type         string     `json:"type,omitempty"`
	NonRetryable bool       `json:"non_retryable,omitempty"`
	Details      []*payload `json:"details,omitempty"`
	Cause        *failure   `json:"cause,omitempty"`
}

func (f *failure) toError() error {
	var cause error
	if f.Cause != nil {
		cause = f.Cause.toError()
	}
	var details []interface{}
	if len(f.Details) > 0 {
		details = []interface{}{payloadsToRaw(f.Details)}
	}
	if f.NonRetryable {
		return temporal.NewNonRetryableApplicationError(f.Message, f.Type, cause, details...)
	}
	return temporal.NewApplicationErrorWithCause(f.Message, f.Type, cause, details...)
}

type info struct {
	Params []*payload `json:"params,omitempty"`
}

type payload struct {
	Metadata map[string][]byte `json:"metadata,omitempty"`
	Data     []byte            `json:"data"`
}

func payloadsFromProto(payloads []*commonpb.Payload) []*payload {
	ret := make([]*payload, len(payloads))
	for i, p := range payloads {
		ret[i].Metadata = p.Metadata
		ret[i].Data = p.Data
	}
	return ret
}

func payloadsFromRaw(raw *converter.RawPayloads) []*payload {
	if raw == nil || len(raw.Payloads) == 0 {
		return nil
	}
	return payloadsFromProto(raw.Payloads)
}

func payloadsToProto(payloads []*payload) []*commonpb.Payload {
	ret := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		ret[i].Metadata = p.Metadata
		ret[i].Data = p.Data
	}
	return ret
}

func payloadsToRaw(payloads []*payload) *converter.RawPayloads {
	return &converter.RawPayloads{Payloads: payloadsToProto(payloads)}
}
