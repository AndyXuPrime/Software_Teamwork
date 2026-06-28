package service

import (
	"io"
	"time"
)

type DocumentStatus string

const (
	DocumentStatusUploaded DocumentStatus = "uploaded"
)

type RequestContext struct {
	RequestID      string
	UserID         string
	Roles          []string
	Permissions    []string
	ForwardedFor   string
	ForwardedProto string
}

type Document struct {
	ID              string
	KnowledgeBaseID string
	Name            string
	Status          DocumentStatus
	Tags            []string
	ErrorMessage    *string
	CreatedAt       time.Time
	ContentType     string
	SizeBytes       int64
	ObjectKey       string
	OwnerUserID     string
	DeletedAt       *time.Time
}

type StoredObject struct {
	Body        io.ReadCloser
	ContentType string
	SizeBytes   int64
}

type DocumentContent struct {
	Document    Document
	Body        io.ReadCloser
	ContentType string
	SizeBytes   int64
}

type UploadDocumentInput struct {
	KnowledgeBaseID string
	FileName        string
	ContentType     string
	SizeBytes       int64
	Tags            []string
	Content         io.Reader
}

type UpdateDocumentInput struct {
	DocumentID string
	Tags       []string
}
