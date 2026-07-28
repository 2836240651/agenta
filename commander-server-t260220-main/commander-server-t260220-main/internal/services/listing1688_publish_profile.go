package services

import "strings"

const (
	listing1688PublishDraftStatusReady         = "ready"
	listing1688PublishDraftStatusMissingFields = "missing_fields"
)

type listing1688PublishWorkflowData struct {
	Title  string
	Images []string
	Skus   []map[string]any
	Fields map[string]any
}

type listing1688PublishStorePreset struct {
	Fields map[string]any
}

type listing1688PublishCategoryTemplate struct {
	CategoryID     string
	Fields         map[string]any
	RequiredFields []string
}

type listing1688PublishProfileInput struct {
	Workflow         listing1688PublishWorkflowData
	StorePreset      listing1688PublishStorePreset
	CategoryTemplate listing1688PublishCategoryTemplate
	UserFields       map[string]any
}

type listing1688PublishDraft struct {
	CategoryID       string         `json:"category_id"`
	FormData         map[string]any `json:"form_data"`
	MissingFields    []string       `json:"missing_fields"`
	ValidationStatus string         `json:"validation_status"`
}

func listing1688BuildPublishDraft(input listing1688PublishProfileInput) listing1688PublishDraft {
	formData := make(map[string]any)
	listing1688MergePublishFields(formData, input.StorePreset.Fields)
	listing1688MergePublishFields(formData, input.CategoryTemplate.Fields)

	if title := strings.TrimSpace(input.Workflow.Title); title != "" {
		formData["title"] = title
	}
	if len(input.Workflow.Images) > 0 {
		formData["images"] = input.Workflow.Images
	}
	if len(input.Workflow.Skus) > 0 {
		formData["skus"] = input.Workflow.Skus
	}
	listing1688MergePublishFields(formData, input.Workflow.Fields)
	listing1688MergePublishFields(formData, input.UserFields)

	missingFields := make([]string, 0)
	seen := make(map[string]struct{})
	for _, field := range input.CategoryTemplate.RequiredFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		if !listing1688PublishFieldPresent(formData[field]) {
			missingFields = append(missingFields, field)
		}
	}

	status := listing1688PublishDraftStatusReady
	if len(missingFields) > 0 {
		status = listing1688PublishDraftStatusMissingFields
	}
	return listing1688PublishDraft{
		CategoryID:       strings.TrimSpace(input.CategoryTemplate.CategoryID),
		FormData:         formData,
		MissingFields:    missingFields,
		ValidationStatus: status,
	}
}

func listing1688MergePublishFields(target, source map[string]any) {
	for key, value := range source {
		key = strings.TrimSpace(key)
		if key == "" || !listing1688PublishFieldPresent(value) {
			continue
		}
		target[key] = value
	}
}

func listing1688PublishFieldPresent(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []string:
		return len(typed) > 0
	case []map[string]any:
		return len(typed) > 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func listing1688PublishDraftCanSubmit(validationStatus string, draftSaved bool) bool {
	return validationStatus == listing1688PublishDraftStatusReady && draftSaved
}
