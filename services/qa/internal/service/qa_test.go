package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
)

type fakeRepository struct {
	conversation   Conversation
	messages       []Message
	messageOptions MessageListOptions
	savedSteps     []ReasoningStep
	savedEvents    []StreamEvent
	invocations    []ModelInvocation
	finalization   ResponseRunFinalization
	run            ResponseRun
}

func (r *fakeRepository) CreateConversation(_ context.Context, value Conversation) (Conversation, error) {
	r.conversation = value
	return value, nil
}
func (r *fakeRepository) ListConversations(_ context.Context, _ string, options ConversationListOptions) (Page[Conversation], error) {
	return Page[Conversation]{Items: []Conversation{r.conversation}, Page: options.Page, PageSize: options.PageSize, Total: 1}, nil
}
func (r *fakeRepository) GetConversation(context.Context, string, string) (Conversation, error) {
	return r.conversation, nil
}
func (r *fakeRepository) UpdateConversation(_ context.Context, _ string, value Conversation) (Conversation, error) {
	r.conversation = value
	return value, nil
}
func (*fakeRepository) DeleteConversation(context.Context, string, string) error { return nil }
func (r *fakeRepository) ListMessages(_ context.Context, _ string, _ string, options MessageListOptions) (Page[Message], error) {
	r.messageOptions = options
	return Page[Message]{Items: append([]Message(nil), r.messages...), Page: options.Page, PageSize: options.PageSize, Total: len(r.messages)}, nil
}
func (r *fakeRepository) AppendMessages(_ context.Context, _, sessionID string, start ResponseRunStart, values ...Message) (ResponseRun, error) {
	r.messages = append(r.messages, values...)
	maxIterations := start.MaxIterations
	if maxIterations == 0 {
		maxIterations = 5
	}
	r.run = ResponseRun{ID: "run-id", SessionID: sessionID, UserMessageID: values[0].ID, AssistantMessageID: values[1].ID, Status: "running", MaxIterations: maxIterations, CreatedAt: values[0].CreatedAt}
	return r.run, nil
}
func (r *fakeRepository) SaveStreamEvents(_ context.Context, _, _ string, events []StreamEvent) error {
	r.savedEvents = append([]StreamEvent(nil), events...)
	return nil
}
func (r *fakeRepository) GetResponseRun(context.Context, string, string) (ResponseRun, error) {
	r.run.Status = "completed"
	return r.run, nil
}
func (r *fakeRepository) UpdateMessage(_ context.Context, _ string, value Message) error {
	for index := range r.messages {
		if r.messages[index].ID == value.ID {
			r.messages[index] = value
			return nil
		}
	}
	return errors.New("message not found")
}
func (r *fakeRepository) FinalizeResponseRun(_ context.Context, _ string, final ResponseRunFinalization) (ResponseRun, error) {
	r.finalization = final
	if err := r.UpdateMessage(context.Background(), "", final.AssistantMessage); err != nil {
		return ResponseRun{}, err
	}
	r.savedSteps = append([]ReasoningStep(nil), final.ReasoningSteps...)
	r.savedEvents = append([]StreamEvent(nil), final.StreamEvents...)
	r.run.Status = final.Status
	r.run.CurrentIteration = final.CurrentIteration
	r.run.TotalTokens = final.TotalTokens
	r.run.CompletedAt = &final.CompletedAt
	if final.TerminationReason != "" {
		reason := final.TerminationReason
		r.run.TerminationReason = &reason
	}
	return r.run, nil
}
func (r *fakeRepository) SaveReasoningSteps(_ context.Context, _, _ string, steps []ReasoningStep) error {
	r.savedSteps = append([]ReasoningStep(nil), steps...)
	return nil
}
func (r *fakeRepository) SaveModelInvocation(_ context.Context, _ string, invocation ModelInvocation) (string, error) {
	r.invocations = append(r.invocations, invocation)
	return fmt.Sprintf("invocation-%d", invocation.IterationNo), nil
}

type fakeAgentRunner struct {
	input []agent.Message
}
type blockingAgentRunner struct{ started chan struct{} }

func (r blockingAgentRunner) RunWithObserver(ctx context.Context, _ []agent.Message, _ agent.Observer) (agent.Result, error) {
	close(r.started)
	<-ctx.Done()
	return agent.Result{}, ctx.Err()
}

func (r *fakeAgentRunner) RunWithObserver(_ context.Context, input []agent.Message, observer agent.Observer) (agent.Result, error) {
	r.input = append([]agent.Message(nil), input...)
	observer(agent.Event{Type: agent.EventModelStarted, Iteration: 1})
	observer(agent.Event{Type: agent.EventModelCompleted, Iteration: 1, Usage: agent.TokenUsage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14}})
	final := agent.Message{Role: agent.RoleAssistant, Content: "测试回答"}
	return agent.Result{Final: final, Messages: append(input, final), Iterations: 1}, nil
}

type errorAgentRunner struct{ err error }

func (r errorAgentRunner) RunWithObserver(_ context.Context, _ []agent.Message, observer agent.Observer) (agent.Result, error) {
	observer(agent.Event{Type: agent.EventModelStarted, Iteration: 1})
	return agent.Result{}, r.err
}

type maxIterationsAgentRunner struct{}

func (maxIterationsAgentRunner) RunWithObserver(_ context.Context, _ []agent.Message, observer agent.Observer) (agent.Result, error) {
	observer(agent.Event{Type: agent.EventModelStarted, Iteration: 2})
	observer(agent.Event{Type: agent.EventModelCompleted, Iteration: 2, Usage: agent.TokenUsage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5}})
	return agent.Result{Iterations: 2}, agent.ErrMaxIterations
}

type fakeRuntimeProvider struct {
	runner         AgentRunner
	prompt         string
	maxIterations  int
	overallTimeout time.Duration
}

func (p fakeRuntimeProvider) Acquire() (RuntimeSnapshot, func(), error) {
	maxIterations := p.maxIterations
	if maxIterations == 0 {
		maxIterations = 5
	}
	return RuntimeSnapshot{
		Runner: p.runner, SystemPrompt: p.prompt, LLMModel: "deepseek-v4-pro", LLMProfileID: "default",
		QAConfigVersionID: "qa-config-id", LLMConfigVersionID: "llm-config-id",
		MaxIterations: maxIterations, OverallTimeout: p.overallTimeout,
	}, func() {}, nil
}

func TestAskPersistsConversationMessagesAndDisplayableSteps(t *testing.T) {
	now := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", OwnerUserID: "user-id", Title: "新对话", Status: "active", CreatedAt: now, UpdatedAt: now}}
	runner := &fakeAgentRunner{}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: runner, prompt: "system prompt"})
	if err != nil {
		t.Fatal(err)
	}
	qa.now = func() time.Time { return now }
	var events []ProgressEvent
	result, err := qa.Ask(context.Background(), "user-id", "conversation-id", AskInput{Message: "锅炉检查要求", Mode: "knowledge_qa"}, func(event ProgressEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AssistantMessage.Content != "测试回答" || result.AssistantMessage.Status != "completed" {
		t.Fatalf("unexpected answer: %+v", result.AssistantMessage)
	}
	if repository.conversation.Title != "锅炉检查要求" {
		t.Fatalf("automatic title = %q", repository.conversation.Title)
	}
	if len(repository.messages) != 2 || repository.messages[1].Content != "测试回答" {
		t.Fatalf("unexpected persisted messages: %+v", repository.messages)
	}
	if len(repository.savedSteps) != 2 || len(events) != 6 || len(repository.savedEvents) != 6 {
		t.Fatalf("steps=%d events=%d", len(repository.savedSteps), len(events))
	}
	if result.ResponseRun.Status != "completed" || result.ResponseRun.TerminationReason == nil || *result.ResponseRun.TerminationReason != "completed" || result.ResponseRun.TotalTokens != 14 {
		t.Fatalf("unexpected response run: %+v", result.ResponseRun)
	}
	if len(repository.invocations) != 1 || repository.invocations[0].TotalTokens != 14 {
		t.Fatalf("unexpected model invocations: %+v", repository.invocations)
	}
	if len(runner.input) < 2 || runner.input[0].Role != agent.RoleSystem || runner.input[len(runner.input)-1].Content != "锅炉检查要求" {
		t.Fatalf("unexpected agent input: %+v", runner.input)
	}
}

func TestAskRejectsUnsupportedDataAnalysis(t *testing.T) {
	err := validateAskInput(AskInput{Message: "分析表格", Mode: "data_analysis"})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeUnsupportedIntent {
		t.Fatalf("error = %v, want unsupported_intent", err)
	}
}

func TestListConversationsNormalizesDocumentedOptions(t *testing.T) {
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", Status: "active"}}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: &fakeAgentRunner{}, prompt: "system"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := qa.ListConversations(context.Background(), "user-id", ConversationListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Page != 1 || result.PageSize != 20 {
		t.Fatalf("page=%d pageSize=%d", result.Page, result.PageSize)
	}
	if _, err = qa.ListConversations(context.Background(), "user-id", ConversationListOptions{Status: "deleted"}); err == nil {
		t.Fatal("expected invalid status to fail")
	}
	if _, err = qa.ListConversations(context.Background(), "user-id", ConversationListOptions{Sort: "title"}); err == nil {
		t.Fatal("expected invalid sort to fail")
	}
}

func TestListMessagesNormalizesDocumentedOptions(t *testing.T) {
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", Status: "active"}}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: &fakeAgentRunner{}, prompt: "system"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = qa.ListMessages(context.Background(), "user-id", "conversation-id", MessageListOptions{IncludeThinking: true, IncludeCitations: true})
	if err != nil {
		t.Fatal(err)
	}
	want := MessageListOptions{Page: 1, PageSize: 50, IncludeThinking: true, IncludeCitations: true}
	if repository.messageOptions != want {
		t.Fatalf("options=%+v want %+v", repository.messageOptions, want)
	}
	if _, err = qa.ListMessages(context.Background(), "user-id", "conversation-id", MessageListOptions{Page: 1, PageSize: 101}); err == nil {
		t.Fatal("expected invalid page size to fail")
	}
	if _, err = qa.ListMessages(context.Background(), "", "conversation-id", MessageListOptions{}); err == nil {
		t.Fatal("expected missing user to fail")
	}
}

func TestCancelActiveRunCancelsAgentAndPersistsCancelledMessage(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", OwnerUserID: "user-id", Status: "active", CreatedAt: now, UpdatedAt: now}}
	runner := blockingAgentRunner{started: make(chan struct{})}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: runner, prompt: "system"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := qa.Ask(context.Background(), "user-id", "conversation-id", AskInput{Message: "cancel me"}, nil)
		done <- err
	}()
	<-runner.started
	qa.CancelActiveRun("run-id")
	if err := <-done; err == nil {
		t.Fatal("expected cancelled ask to fail")
	}
	if got := repository.messages[1].Status; got != "cancelled" {
		t.Fatalf("assistant status=%q", got)
	}
	if repository.finalization.TerminationReason != "cancelled" || repository.finalization.Status != "cancelled" {
		t.Fatalf("finalization=%+v", repository.finalization)
	}
}

func TestAskPersistsModelFailureReason(t *testing.T) {
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", OwnerUserID: "user-id", Status: "active"}}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: errorAgentRunner{err: errors.New("provider secret detail")}, prompt: "system"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = qa.Ask(context.Background(), "user-id", "conversation-id", AskInput{Message: "hello"}, nil)
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeDependency {
		t.Fatalf("error=%v, want dependency_error", err)
	}
	if repository.finalization.Status != "failed" || repository.finalization.TerminationReason != "model_error" {
		t.Fatalf("finalization=%+v", repository.finalization)
	}
	if len(repository.invocations) != 1 || repository.invocations[0].Status != "failed" || repository.invocations[0].ErrorMessage != "answer generation failed" {
		t.Fatalf("invocations=%+v", repository.invocations)
	}
}

func TestAskPersistsTimeoutReason(t *testing.T) {
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", OwnerUserID: "user-id", Status: "active"}}
	runner := blockingAgentRunner{started: make(chan struct{})}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: runner, prompt: "system", overallTimeout: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	_, err = qa.Ask(context.Background(), "user-id", "conversation-id", AskInput{Message: "timeout"}, nil)
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeDependency {
		t.Fatalf("error=%v, want dependency_error", err)
	}
	if repository.finalization.Status != "failed" || repository.finalization.TerminationReason != "timeout" {
		t.Fatalf("finalization=%+v", repository.finalization)
	}
}

func TestAskPersistsMaxIterationsReason(t *testing.T) {
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", OwnerUserID: "user-id", Status: "active"}}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: maxIterationsAgentRunner{}, prompt: "system", maxIterations: 2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = qa.Ask(context.Background(), "user-id", "conversation-id", AskInput{Message: "loop"}, nil)
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeDependency {
		t.Fatalf("error=%v, want dependency_error", err)
	}
	if repository.finalization.Status != "failed" || repository.finalization.TerminationReason != "max_iterations" || repository.finalization.CurrentIteration != 2 {
		t.Fatalf("finalization=%+v", repository.finalization)
	}
	if len(repository.invocations) == 0 || repository.invocations[0].TotalTokens != 5 {
		t.Fatalf("invocations=%+v", repository.invocations)
	}
}
