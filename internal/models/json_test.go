package models

import (
	"database/sql/driver"
	"reflect"
	"testing"
)

func TestJSON_Value(t *testing.T) {
	tests := []struct {
		name    string
		j       JSON
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "empty JSON",
			j:       JSON([]byte{}),
			want:    nil,
			wantErr: false,
		},
		{
			name:    "valid JSON object",
			j:       JSON([]byte(`{"key":"value"}`)),
			want:    `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "valid JSON array",
			j:       JSON([]byte(`[1,2,3]`)),
			want:    `[1,2,3]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.j.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSON.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSON_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    JSON
		wantErr bool
	}{
		{
			name:    "nil value",
			value:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "byte slice",
			value:   []byte(`{"key":"value"}`),
			want:    JSON(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "string",
			value:   `{"key":"value"}`,
			want:    JSON(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "invalid type",
			value:   123,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var j JSON
			err := j.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(j, tt.want) {
				t.Errorf("JSON.Scan() = %v, want %v", j, tt.want)
			}
		})
	}
}

func TestJSON_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		j       JSON
		want    []byte
		wantErr bool
	}{
		{
			name:    "nil JSON",
			j:       nil,
			want:    []byte("null"),
			wantErr: false,
		},
		{
			name:    "valid JSON",
			j:       JSON(`{"key":"value"}`),
			want:    []byte(`{"key":"value"}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.j.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSON.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
