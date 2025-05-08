package models

// --- SillyTavern V2 Character Card Structures ---
type CharacterCardV2 struct {
	Spec        string     `json:"spec"`
	SpecVersion string     `json:"spec_version"`
	Data        CardData   `json:"data"`
	Extensions  Extensions `json:"extensions,omitempty"`
}
type CardData struct {
	Name                    string     `json:"name"`
	Description             string     `json:"description"`
	Personality             string     `json:"personality"`
	Scenario                string     `json:"scenario"`
	FirstMes                string     `json:"first_mes"`
	MesExample              string     `json:"mes_example"`
	CreatorNotes            string     `json:"creator_notes,omitempty"`
	SystemPrompt            string     `json:"system_prompt,omitempty"`
	PostHistoryInstructions string     `json:"post_history_instructions,omitempty"`
	AlternateGreetings      []string   `json:"alternate_greetings,omitempty"`
	Tags                    []string   `json:"tags,omitempty"`
	Creator                 string     `json:"creator,omitempty"`
	CharacterVersion        string     `json:"character_version,omitempty"`
	CharacterBook           *Lorebook  `json:"character_book,omitempty"` // Embedded lorebook
	VisualDescription       string     `json:"visual_description,omitempty"`
	ThoughtPattern          string     `json:"thought_pattern,omitempty"`
	SpeechPattern           string     `json:"speech_pattern,omitempty"`
	Relationships           string     `json:"relationships,omitempty"` // Summarized relationships
	Goals                   string     `json:"goals,omitempty"`
	Fears                   string     `json:"fears,omitempty"`
	Strengths               string     `json:"strengths,omitempty"`
	Weaknesses              string     `json:"weaknesses,omitempty"`
	Alignment               string     `json:"alignment,omitempty"`
	Tropes                  []string   `json:"tropes,omitempty"`
	Extensions              Extensions `json:"extensions,omitempty"` // For arbitrary data
}
type Lorebook struct {
	Name              string          `json:"name,omitempty"`
	Description       string          `json:"description,omitempty"`
	ScanDepth         int             `json:"scan_depth,omitempty"`
	TokenBudget       int             `json:"token_budget,omitempty"`
	RecursiveScanning bool            `json:"recursive_scanning,omitempty"`
	InsertionOrder    int             `json:"insertion_order"`
	Enabled           bool            `json:"enabled"`
	Entries           []LorebookEntry `json:"entries"`
	Extensions        Extensions      `json:"extensions,omitempty"`
}
type LorebookEntry struct {
	Keys           []string   `json:"keys"`
	Content        string     `json:"content"`
	Enabled        bool       `json:"enabled"`
	InsertionOrder int        `json:"insertion_order"`
	Priority       int        `json:"priority,omitempty"`
	Comment        string     `json:"comment,omitempty"`
	SelectiveLogic string     `json:"selectiveLogic,omitempty"`
	SecondaryKeys  []string   `json:"secondaryKeys,omitempty"`
	Constant       bool       `json:"constant,omitempty"`
	CaseSensitive  bool       `json:"case_sensitive,omitempty"`
	Probability    int        `json:"probability,omitempty"`
	Extensions     Extensions `json:"extensions,omitempty"`
}
type Extensions map[string]interface{}

// --- Request and Response Payloads ---

// Struct for AI-suggested tool details (Option 4)
type AISuggestedTool struct {
	ToolType           string `json:"tool_type"`
	ToolName           string `json:"tool_name"`
	ToolJustification string `json:"tool_justification"`
}

type RequestPayload struct {
	APIKey          string `json:"apiKey"`
	Series          string `json:"series"`
	Option          string `json:"option"`
	Model           string `json:"model"`
	ToolCardPurpose string `json:"toolCardPurpose,omitempty"` // New field for Option 3
}
type ResponsePayload struct {
	Series           string `json:"series"`
	OptionChosen     string `json:"option_chosen"`
	ModelUsed        string `json:"model_used"`
	APIKeyReceived   bool   `json:"api_key_received"`
	Message          string `json:"message"`
	GeneratedContent string `json:"generated_content,omitempty"`
	Timestamp        string `json:"timestamp"`
	Error            string `json:"error,omitempty"`
	LogIdentifier    string `json:"log_identifier,omitempty"`
}

// --- Constants ---
const CHARACTER_CARD_SEPARATOR = "CHARACTER_CARD_SEPARATOR_AI_FICTION_FORGE"

