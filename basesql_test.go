package basesql

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid tenant auth config",
			config: &Config{
				AppID:     "cli_test_app_id",
				AppSecret: "test_app_secret_12345678",
				AppToken:  "test_app_token",
				AuthType:  AuthTypeTenant,
			},
			wantErr: false,
		},
		{
			name: "valid user auth config",
			config: &Config{
				AppID:       "cli_test_app_id",
				AppSecret:   "test_app_secret_12345678",
				AppToken:    "test_app_token",
				AuthType:    AuthTypeUser,
				AccessToken: "u-test",
			},
			wantErr: false,
		},
		{
			name: "missing app_id",
			config: &Config{
				AppSecret: "test_app_secret",
				AppToken:  "test_app_token",
				AuthType:  AuthTypeTenant,
			},
			wantErr: true,
		},
		{
			name: "missing app_secret",
			config: &Config{
				AppID:    "cli_test_app_id",
				AppToken: "test_app_token",
				AuthType: AuthTypeTenant,
			},
			wantErr: true,
		},
		{
			name: "missing app_token",
			config: &Config{
				AppID:     "cli_test_app_id",
				AppSecret: "test_app_secret_12345678",
				AuthType:  AuthTypeTenant,
			},
			wantErr: true,
		},
		{
			name: "user auth missing access_token",
			config: &Config{
				AppID:     "cli_test_app_id",
				AppSecret: "test_app_secret_12345678",
				AppToken:  "test_app_token",
				AuthType:  AuthTypeUser,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFieldTypeMapping(t *testing.T) {
	// 测试所有定义的字段类型都有对应的映射
	definedFieldTypes := []FieldType{
		FieldTypeText,
		FieldTypeNumber,
		FieldTypeSingleSelect,
		FieldTypeMultiSelect,
		FieldTypeDate,
		FieldTypeCheckbox,
		FieldTypeUser,
		FieldTypePhone,
		FieldTypeURL,
		FieldTypeAttachment,
		FieldTypeBarcode,
		FieldTypeProgress,
		FieldTypeCurrency,
		FieldTypeRating,
		FieldTypeFormula,
		FieldTypeLookup,
		FieldTypeCreatedTime,
		FieldTypeModifiedTime,
		FieldTypeCreatedUser,
		FieldTypeModifiedUser,
		FieldTypeAutoNumber,
	}

	for _, fieldType := range definedFieldTypes {
		if _, ok := FieldTypeMapping[fieldType]; !ok {
			t.Errorf("FieldType %d missing in FieldTypeMapping", fieldType)
		}
	}

	// 反向测试：确保映射中的所有类型都在定义列表中
	for fieldType := range FieldTypeMapping {
		found := false
		for _, definedType := range definedFieldTypes {
			if fieldType == definedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FieldType %d in FieldTypeMapping but not in defined types", fieldType)
		}
	}
}

func TestField_ConvertToGoValue(t *testing.T) {
	tests := []struct {
		name      string
		fieldType FieldType
		input     interface{}
		expected  interface{}
	}{
		{
			name:      "text field",
			fieldType: FieldTypeText,
			input:     "hello",
			expected:  "hello",
		},
		{
			name:      "number field",
			fieldType: FieldTypeNumber,
			input:     42.0,
			expected:  42.0,
		},
		{
			name:      "checkbox field true",
			fieldType: FieldTypeCheckbox,
			input:     true,
			expected:  true,
		},
		{
			name:      "checkbox field false",
			fieldType: FieldTypeCheckbox,
			input:     false,
			expected:  false,
		},
		{
			name:      "date field timestamp",
			fieldType: FieldTypeDate,
			input:     int64(1640995200000), // 2022-01-01 00:00:00 UTC
			expected:  time.Unix(1640995200, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &Field{Type: tt.fieldType}
			result := field.ConvertToGoValue(tt.input)

			// 对于时间类型，需要特殊比较
			if tt.fieldType == FieldTypeDate {
				expectedTime := tt.expected.(time.Time)
				resultTime := result.(time.Time)
				if !expectedTime.Equal(resultTime) {
					t.Errorf("Field.ConvertToGoValue() = %v, expected %v", result, tt.expected)
				}
			} else if result != tt.expected {
				t.Errorf("Field.ConvertToGoValue() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestField_ConvertFromGoValue(t *testing.T) {
	tests := []struct {
		name      string
		fieldType FieldType
		input     interface{}
		expected  interface{}
	}{
		{
			name:      "text field",
			fieldType: FieldTypeText,
			input:     "hello",
			expected:  "hello",
		},
		{
			name:      "number field int",
			fieldType: FieldTypeNumber,
			input:     42,
			expected:  42.0,
		},
		{
			name:      "checkbox field",
			fieldType: FieldTypeCheckbox,
			input:     true,
			expected:  true,
		},
		{
			name:      "date field",
			fieldType: FieldTypeDate,
			input:     time.Unix(1640995200, 0),
			expected:  int64(1640995200000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &Field{Type: tt.fieldType}
			result := field.ConvertFromGoValue(tt.input)
			if result != tt.expected {
				t.Errorf("Field.ConvertFromGoValue() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDialector_Name(t *testing.T) {
	dialector := &Dialector{}
	if dialector.Name() != "basesql" {
		t.Errorf("Dialector.Name() = %v, expected %v", dialector.Name(), "basesql")
	}
}

func TestErrorTypes(t *testing.T) {
	// 测试错误类型
	err := ErrInvalidConfig("test details")
	if err == nil {
		t.Error("ErrInvalidConfig should return an error")
	}

	baseErr, ok := err.(*BaseError)
	if !ok {
		t.Error("ErrInvalidConfig should return a BaseError")
	}

	if baseErr.Code != "INVALID_CONFIG" {
		t.Errorf("Expected error code INVALID_CONFIG, got %s", baseErr.Code)
	}

	if baseErr.Details != "test details" {
		t.Errorf("Expected error details 'test details', got %s", baseErr.Details)
	}
}
