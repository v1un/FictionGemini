package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template" // Added for prompt templating
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const toolCardPromptTemplate = `
Generate a SillyTavern V2 Character Card JSON specifically designed as a UTILITY or TOOL card for the series \\\'{{.SeriesName}}\\\'.
The primary purpose of this card is: \\\'{{.ToolPurpose}}\\\'. This tool should feel like an authentic part of the \\\'{{.SeriesName}}\\\' world.

Your ENTIRE response MUST be ONLY a single, valid JSON object, starting with \'{\' and ending with \'}\'. No other text, comments, explanations, or markdown formatting should precede or follow this JSON object.
The JSON object must strictly adhere to the SillyTavern V2 Character Card specification:
  "spec": "chara_card_v2",
  "spec_version": "2.0".

The "data" object must be meticulously crafted for this tool's function, infused with the flavor of \'{{.SeriesName}}\':

1.  "name": "{{.ToolPurpose}} of {{.SeriesName}}" (Make this concise, descriptive, and thematically appropriate for \'{{.SeriesName}}\')
2.  "description":
    This field is CRUCIAL. It will store the ACTUAL DATA for the tool in a clear, human-readable, structured format, styled to fit \'{{.SeriesName}}\'.
    Initialize it with a sensible default or empty state appropriate for \'{{.ToolPurpose}}\'.
    When initializing data, use thematic placeholders or examples *drawn from the lore of \'{{.SeriesName}}\'* if appropriate.
    Use Unicode box-drawing characters (like ╔═╗, ║, ╚═╝, ╠═╣, ╣, ╩, ╦) to create panels, sections, and tables for the data. Use Markdown for lists within these panels if appropriate.
    Ensure consistent alignment and spacing to maintain a clean, readable interface. Thematic emojis (💰, 📜, ⚔️) can be used sparingly.
    Examples:
      - If \'{{.ToolPurpose}}\' is "Player Character Stats" for a gritty fantasy series \'{{.SeriesName}}\':
        "╔═════════════════════════════╗\\n║   STATISTICS LEDGER ({{user}})  ║\\n╠═════════════════╦═════════════╣\\n║ Reputation      ║ Unknown Scrivener ║\\n║ Class           ║ Uninitiated     ║\\n║ Level           ║ 1 (Novice)      ║\\n╠═════════════════╬═════════════╣\\n║ Vitality (HP)   ║ 10/10           ║\\n║ Essence (MP)    ║ 5/5             ║\\n╠═════════════════╬═════════════╣\\n║ Might (STR)     ║ 10              ║\\n║ Agility (DEX)   ║ 10              ║\\n║ Stamina (CON)   ║ 10              ║\\n║ Intellect (INT) ║ 10              ║\\n║ Wisdom (WIS)    ║ 10              ║\\n║ Presence (CHA)  ║ 10              ║\\n╠═════════════════╬═════════════╣\\n║ Coin (Gold)     ║ 0 Copper Bits   ║\\n║ Burdens/Boons   ║ None of note    ║\\n╚═════════════════╩═════════════╝"
      - If \'{{.ToolPurpose}}\' is "Party Inventory" for a high magic series \'{{.SeriesName}}\':
        "╔═══════════════════════════════════╗\\n║ ✨ Shared Party Satchel (Enchanted) ✨ ║\\n╠═══════════════════════════════════╣\\n║ - Slot 1: [Common Alchemical Concoction]║\\n║ - Slot 2: [Minor Enchanted Trinket]   ║\\n║ - Slot 3: Empty                       ║\\n╠═══════════════════════════════════╣\\n║ 💰 Party Treasury: 0 Lumina Shards    ║\\n╚═══════════════════════════════════╝"
    Use newlines (\\\\n) for formatting. The AI (as this tool card) will be instructed to "rewrite" this description to reflect updates, maintaining the established style and GUI structure.

3.  "personality":
    Describe the tool's "persona" or operational style, ensuring it subtly reflects the dominant tone and themes of \'{{.SeriesName}}\'.
    Examples:
      - For \'{{.SeriesName}}\' (Dark Fantasy): "A grim, factual magical ledger, its script appearing in blood-red ink. It records all entries with cold, unwavering precision. Offers no commentary, only data."
      - For \'{{.SeriesName}}\' (Sci-Fi Adventure): "A chirpy, slightly sarcastic AI assistant integrated into your neural implant. Provides data updates with occasional unsolicited \'helpful\' advice or commentary on your questionable choices."

4.  "scenario":
    A brief statement setting the context for using this tool, grounded in the world of \'{{.SeriesName}}\'.
    Example: "This is the {{char}}, a specialized {{.ToolPurpose}} mechanism from the world of \'{{.SeriesName}}\'. It is designed to aid you in tracking vital information during your endeavors. You can interact with it using clear commands to view, add, remove, or update the recorded data presented in its text-GUI."

5.  "first_mes":
    The initial message the tool card sends. It should introduce itself, state its purpose, show the initial data state (by reproducing the formatted GUI from the \'description\' field), and give clear examples of how to interact with it, using language appropriate for \'{{.SeriesName}}\'.
    Example for "Player Character Stats" in \'{{.SeriesName}}\' (Gritty Fantasy):
    "Hark, {{user}}! I am the {{char}}, your steadfast Chronicler of Deeds for these perilous lands of \'{{.SeriesName}}\'.\\\\nMy ledger currently reads:\\\\n╔═════════════════════════════╗\\\\n║   STATISTICS LEDGER ({{user}})  ║\\\\n╠═════════════════╦═════════════╣\\\\n║ Vitality (HP)   ║ 10/10           ║\\\\n║ ... (other stats) ...         ║\\\\n║ Coin (Gold)     ║ 0 Copper Bits   ║\\\\n╚═════════════════╩═════════════╝\\\\nTo amend your record, speak plainly. For instance: \'Set Vitality to 8/10\' or \'Add 50 Copper Bits to Coin\'. To review your full ledger, command \'Show my chronicle\'."

6.  "mes_example":
    Provide AT LEAST THREE diverse and detailed example dialogues. Each MUST start with "<START>". These examples are CRITICAL for teaching the AI how to behave as this tool, including maintaining the thematic style of \'{{.SeriesName}}\' and the GUI formatting.
    The {{char}}\'s responses after an update MUST explicitly show the *updated section* of the data from the \'description\' (or the full state if small), by **re-rendering the relevant GUI panel or section**, including all Unicode box characters and Markdown.
    Show examples of:
      a. Querying data (e.g., "{{user}}: How stands my Vitality, Chronicler?\\n{{char}}: (The script on the ancient ledger shifts) Your Vitality, {{user}}, is recorded thusly:\\n╔═════════════════╗\\n║ Vitality (HP)   ║ 10/10           ║\\n╚═════════════════╝")
      b. Updating data (e.g., "{{user}}: Scribe, etch my Might as 12.\\n{{char}}: (Quill scratches against parchment) As you command. Might is now 12.\\n╔═════════════════╗\\n║ Might (STR)     ║ 12              ║\\n╚═════════════════╝")
      c. Adding data (if applicable, e.g., for inventory/quest in \'{{.SeriesName}}\'): "{{user}}: Add \'Elixir of Foxglove (x2)\' to the satchel.\\n{{char}}: (The satchel seems to sigh contentedly) \'Elixir of Foxglove (x2)\' now rests within the party\'s shared satchel.\\n╔═══════════════════════════════════╗\\n║ ✨ Shared Party Satchel (Enchanted) ✨ ║\\n╠═══════════════════════════════════╣\\n║ - Elixir of Foxglove (x2)         ║\\n║ - [Minor Enchanted Trinket]       ║\\n╚═══════════════════════════════════╝"

7.  "creator_notes":
    "This card functions as an interactive data management tool, themed for \'{{.SeriesName}}\', presenting its data as a text-based GUI. The AI should interpret user commands to modify and display data stored primarily within its \'description\' field. Focus on parsing user intent for CRUD-like operations and reflecting changes by re-stating/re-rendering the relevant parts of the GUI description. Not a traditional RP character but a stylized interface."

8.  "system_prompt":
    "You are {{char}}, a specialized utility tool meticulously designed for \'{{.ToolPurpose}}\' within the unique world of \'{{.SeriesName}}\'. Your entire persona, method of communication, and the way you present data (as a text-GUI using Unicode box characters and Markdown) should be deeply infused with the style and atmosphere of \'{{.SeriesName}}\'. Your primary function is to manage and display data based on user commands, acting as an authentic in-world interface.\n    When the user issues a command (e.g., \'Set HP to 7\', \'Add 2 Elven Waybreads\'):\n    1. Understand the user\'s intent (query, add, update, delete) through the lens of \'{{.SeriesName}}\' terminology where appropriate.\n    2. If it\'s an update/add/delete, mentally modify the relevant data points stored in your \'description\' field (which is formatted as a text GUI).\n    3. In your response, ALWAYS confirm the action taken, using language fitting your persona and \'{{.SeriesName}}\'.\n    4. Then, clearly present the NEW, UPDATED state of the specific data that was changed by **re-rendering the relevant GUI panel or section from your \'description\'**, including all Unicode box characters and Markdown, to show the change. Quote it directly if possible.\n    5. If the user asks to see data, retrieve it from your \'description\' and present the relevant GUI panel clearly and thematically.\n    Example Update Interaction for a \'{{.SeriesName}}\' styled inventory tool:\n    User: \'Place 3 Sunstone Shards into the relic coffer.\'\n    You ({{char}}): (The ancient coffer glows briefly) \'Understood. Three Sunstone Shards have been secured within the relic coffer.\'\\n╔═════════════════════════════╗\\n║      ✨ RELIC COFFER ✨       ║\\n╠═════════════════════════════╣\\n║ - Sunstone Shards (x3)      ║\\n╚═════════════════════════════╝\' (And you\'d internally update the coffer\'s contents in your description text to this new GUI state).\n    Be precise, thematic, and act as an efficient, in-world data interface that visually updates its GUI."

9.  "post_history_instructions":
    "Always refer to the most recent state of the data in your \'description\' (your text GUI) before making an update. Ensure your responses reflect the cumulative changes from the conversation, maintaining the thematic consistency of \'{{.SeriesName}}\' and the GUI structure. If the user asks for the current state, ensure you provide the absolute latest version of the data you are tracking, presented in your established in-world, GUI-formatted style."

10. "tags": ["Tool", "Utility", "{{.SeriesName}}", "{{.ToolPurpose}}", "Data Tracker", "Thematic Interface", "Text GUI", "AI Generated"]
11. "creator": "AI Fiction Forge (Tool Mode v1.2 - GUI Enhanced)"
12. "character_version": "1.2T"
// The \'character_book\' field is intentionally NOT required for this Tool card.

Do NOT include any text, comments, or markdown formatting outside the main, single JSON object.
The entire response MUST be a single, complete, and valid JSON object.
Be creative in infusing the \'{{.SeriesName}}\' theme into the tool\'s data structure, personality, and interaction style, ensuring it remains functional and its data is clearly structured in the description and presented as a text-GUI.
`

const CHARACTER_CARD_SEPARATOR = "CHARACTER_CARD_SEPARATOR_AI_FICTION_FORGE"

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
	APIKey string `json:"apiKey"`
	Series string `json:"series"`
	Option string `json:"option"`
	Model  string `json:"model"`
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

// --- File System Storage Helpers ---
// sanitizeStringForPath cleans a string to be file system friendly.
func sanitizeStringForPath(input string, makeLower bool) string {
	if makeLower {
		input = strings.ToLower(input)
	}
	input = strings.ReplaceAll(input, " ", "_")

	// Allow basic alphanumeric, underscore, hyphen, period. Remove others.
	reg := regexp.MustCompile("[^a-zA-Z0-9_\\-\\.]+")
	input = reg.ReplaceAllString(input, "")

	maxLen := 50 // Max length for a path component
	if len(input) > maxLen {
		input = input[:maxLen]
	}
	if input == "" {
		return "unnamed" // Default if input becomes empty
	}
	return input
}

// saveJSONToFile saves the jsonData to a file within a specific session's logIdentifier directory.
func saveJSONToFile(seriesName, itemType, itemName, logIdentifier string, jsonData []byte) (string, error) {
	// Clean the item name for use as a filename component
	cleanedItemName := sanitizeStringForPath(itemName, false)  // Keep case for item name if meaningful
	if cleanedItemName == "unnamed" || cleanedItemName == "" { // Fallback if item name is not useful
		cleanedItemName = sanitizeStringForPath(itemType, true) // Use item type as name
		if cleanedItemName == "unnamed" {                       // Ultimate fallback
			cleanedItemName = "data"
		}
	}

	baseDir := "jsons"
	// The specificDir is determined by logIdentifier, ensuring all files for this session go into the same folder.
	specificDir := filepath.Join(baseDir, logIdentifier)

	if err := os.MkdirAll(specificDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", specificDir, err)
	}

	// Construct a unique filename using itemType and cleanedItemName
	fileName := fmt.Sprintf("%s_%s.json", sanitizeStringForPath(itemType, true), cleanedItemName)
	fullPath := filepath.Join(specificDir, fileName)

	if err := os.WriteFile(fullPath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON to file %s: %w", fullPath, err)
	}

	log.Printf("Successfully saved JSON to: %s", fullPath)
	return fullPath, nil
}

// sendJSONError sends a structured JSON error response to the client.
func sendJSONError(w http.ResponseWriter, message string, statusCode int, logIdentifier string) {
	log.Printf("Sending error response: %d - %s (Log ID: %s)", statusCode, message, logIdentifier)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// Include logIdentifier in error response for user reference
	json.NewEncoder(w).Encode(ResponsePayload{Error: message, LogIdentifier: logIdentifier, Timestamp: time.Now().Format(time.RFC3339)})
}

// callGeminiAI handles the interaction with the Gemini API.
func callGeminiAI(ctx context.Context, client *genai.GenerativeModel, prompt string) (string, error) {
	logPrompt := prompt
	if len(logPrompt) > 600 { // Increased log truncation for more context
		logPrompt = logPrompt[:600] + "..."
	}
	log.Printf("Sending prompt to Gemini model (truncated if long): %s\n", logPrompt)

	// Generate content
	resp, err := client.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini API call failed: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("gemini API returned nil response")
	}

	// Aggregate text from all parts of all candidates
	var fullResponseTextBuilder strings.Builder
	hasContent := false
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if textPart, ok := part.(genai.Text); ok {
					fullResponseTextBuilder.WriteString(string(textPart))
					hasContent = true
				}
			}
		}
	}

	// Check if any content was actually generated
	if !hasContent {
		var finishReasonMsg strings.Builder
		if len(resp.Candidates) > 0 {
			candidate := resp.Candidates[0] // Check first candidate for more details
			if candidate.FinishReason != genai.FinishReasonStop && candidate.FinishReason != genai.FinishReasonUnspecified {
				finishReasonMsg.WriteString(fmt.Sprintf(" Finish Reason: %s.", candidate.FinishReason.String()))
			}
			// Log safety ratings if present
			if candidate.SafetyRatings != nil {
				for _, sr := range candidate.SafetyRatings {
					if sr.Blocked {
						finishReasonMsg.WriteString(fmt.Sprintf(" Blocked by SafetyRating: Category %s, Probability %s.", sr.Category.String(), sr.Probability.String()))
					}
				}
			}
		}
		// Log Prompt Feedback if available (provides info on why a prompt might have been blocked)
		if resp.PromptFeedback != nil {
			if resp.PromptFeedback.BlockReason != genai.BlockReasonUnspecified {
				finishReasonMsg.WriteString(fmt.Sprintf(" Prompt Feedback Block Reason: %s.", resp.PromptFeedback.BlockReason.String()))
			}
			// You can also log resp.PromptFeedback.SafetyRatings here if needed
		}
		log.Printf("Gemini API returned no textual content or was blocked.%s\n", finishReasonMsg.String())
		return "", fmt.Errorf("gemini API returned no textual content or was blocked.%s", finishReasonMsg.String())
	}

	rawResponse := fullResponseTextBuilder.String()
	aiResponse := strings.TrimSpace(rawResponse) // Basic trim

	// More robust JSON extraction:
	// This regex attempts to find JSON objects {.*} or arrays [.*]
	// that might be wrapped in markdown code blocks (```json ... ``` or ``` ... ```)
	// or might just be plain JSON in the string.
	// It's non-greedy for the content within braces/brackets to handle nested structures better.
	jsonBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\}|\\[.*?\\])\\s*```|(\\{.*?\\}|\\[.*?\\])")
	allMatches := jsonBlockRegex.FindAllStringSubmatch(aiResponse, -1)

	if len(allMatches) > 0 {
		// Prefer the content of a markdown block if found, otherwise take the first plain JSON match
		if allMatches[0][1] != "" { // Content from markdown block
			aiResponse = strings.TrimSpace(allMatches[0][1])
		} else if allMatches[0][2] != "" { // Content from plain JSON match
			aiResponse = strings.TrimSpace(allMatches[0][2])
		}
		// If multiple JSON objects are found (e.g. not correctly formatted as one object by AI),
		// this will pick the first one. The prompt strongly requests a single JSON object.
	} else {
		// If regex doesn't find a clear JSON structure, log a warning but proceed with the trimmed response.
		// This might happen if the AI returns text that isn't valid JSON or is poorly formatted.
		log.Printf("Could not reliably extract a JSON object/array using regex. Proceeding with trimmed raw response. This may cause unmarshalling errors. Raw (trimmed) response (truncated if long): %s\n", aiResponse[:min(600, len(aiResponse))])
	}

	logResponse := aiResponse
	if len(logResponse) > 600 {
		logResponse = logResponse[:600] + "..."
	} // Increased log truncation
	log.Printf("Received (cleaned for JSON extraction) response from Gemini (truncated if long): %s\n", logResponse)
	return aiResponse, nil
}

// --- Main Handler ---
func generateContentHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for /generate")
	var requestPayload RequestPayload
	// Decode request payload
	if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
		// For decode error, logIdentifier might not be available yet, so pass a generic one or empty
			sendJSONError(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest, "decode_error")
			return
		}
		// Note: r.Body is automatically closed by the server after the handler returns.
		// If you need to close it earlier for some reason, you can use defer r.Body.Close()
		// but it's generally not necessary for simple cases like this.
	defer r.Body.Close()

	// Generate a unique identifier for this entire generation session (for file saving).
	// This ensures all files from one "Forge My Fiction!" click go into the same subfolder.
	timestamp := time.Now().Format("20060102_150405.000") // Millisecond precision for uniqueness
	logIdentifier := fmt.Sprintf("%s_%s", sanitizeStringForPath(requestPayload.Series, true), timestamp)

	log.Printf("Processing Series: '%s', Option: %s, Model: %s, Log ID: %s\n", requestPayload.Series, requestPayload.Option, requestPayload.Model, logIdentifier)

	// Validate essential inputs
	if strings.TrimSpace(requestPayload.APIKey) == "" || strings.TrimSpace(requestPayload.Model) == "" || strings.TrimSpace(requestPayload.Series) == "" {
		sendJSONError(w, "Missing API Key, Model, or Series name", http.StatusBadRequest, logIdentifier)
		return
	}
	if requestPayload.Option == "3" && strings.TrimSpace(requestPayload.ToolCardPurpose) == "" {
		sendJSONError(w, "Missing Tool Card Purpose for Option 3", http.StatusBadRequest, logIdentifier)
		return
	}

	// Create context with timeout for the entire generation process
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute) // Increased timeout for complex multi-step generations
	defer cancel()

	// Initialize Gemini client
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(requestPayload.APIKey))
	if err != nil {
		log.Printf("Failed to create Gemini client: %v", err)
		sendJSONError(w, "Failed to initialize AI model connection. Check API Key and network.", http.StatusInternalServerError, logIdentifier)
		return
	}
	defer geminiClient.Close()

	// Select the model specified in the request
	model := geminiClient.GenerativeModel(requestPayload.Model)
	// Potentially set ResponseMIMEType if all chosen models support it reliably.
	// For now, relying on strong prompting for JSON.
	// model.GenerationConfig.ResponseMIMEType = "application/json"

	var optionText, generatedJSONString, finalMessage string

	switch requestPayload.Option {
	case "1": // Lorebook only - With primary characters, secondary characters, locations, events, factions, relationships and etc.
		optionText = "Lorebook Only (Comprehensive)"
		finalMessage = fmt.Sprintf("Processing Option 1: Comprehensive Lorebook for '%s'.\n", requestPayload.Series)

		// Prompt for generating a comprehensive lorebook
		prompt := fmt.Sprintf(`
Generate an EXHAUSTIVELY detailed and extraordinarily comprehensive SillyTavern V2 Lorebook JSON for the series '%s'.
This lorebook must serve as an unparalleled world bible, a rich repository of deep world knowledge, leaving no stone unturned. Cover every conceivable aspect, from grand overarching themes to minute, easily overlooked micro-details.
Your ENTIRE response MUST be ONLY a single, valid JSON object, starting with '{' and ending with '}'. No other text, comments, explanations, or markdown formatting should precede or follow this JSON object.
The JSON object must strictly adhere to the SillyTavern V2 Lorebook specification.

Lorebook Root Structure:
  "name": "Deep Dive Lore for %s",
  "description": "An exceptionally profound and in-depth collection of lore for the world of '%s', meticulously detailing primary and secondary characters, every significant location, pivotal past and obscure historical events, major and minor factions, intricate relationships, core world-building elements (magic systems, technologies, cosmologies, deities), unique flora and fauna, cultural nuances, economic structures, and all other facets that define this universe. This lorebook aims for exhaustive detail.",
  "scan_depth": 35, // Increased scan depth for better contextual understanding.
  "token_budget": 5000, // Increased token budget for richer entries.
  "insertion_order": 0,
  "enabled": true,
  "recursive_scanning": true, // Enable for potential deeper connections.

It MUST contain at least 30-45 (aim for 40+) distinct and EXTREMELY richly detailed 'entries'. Prioritize depth and breadth of information. Explicitly state or hint at interconnections between entries.

1.  PRIMARY CHARACTERS (At least 4-6 entries):
    For each central protagonist, antagonist, or pivotal figure essential to the main plot:
      - "comment": "Primary Character: [Character Name] - [Detailed Role, e.g., 'The Reluctant Hero of Prophecy', 'The Shadow Chancellor pulling the strings']",
      - "content": "Exhaustive profile including: Full appearance (vivid imagery of physical features, attire, distinguishing marks, common expressions), comprehensive personality (core traits, internal conflicts, psychological depth, values, fears, ambitions, motivations, evolution over time), detailed background and history (origin, lineage, key life events, formative experiences, how they became significant, defining quotes or actions), their overarching role and profound impact on the narrative, how they are perceived by different groups/factions, their possessions of note (weapons, artifacts, signature items with their own backstories if relevant). Include a detailed analysis of their most critical relationships (allies, enemies, family, romantic interests - describe the history, nature, power dynamics, and emotional impact of these bonds). What are their signature abilities or skills? What are their profound strengths and crippling weaknesses? What is their moral code or philosophy, and how has it been tested?",
      - "keys": JSON array of 6-8 highly relevant and specific string keywords (e.g., ["[Full Name]", "[Common Name/Alias]", "[Character Title/Role]", "[Key Trait/Internal Conflict]", "[Primary Goal Keyword]", "[Key Relationship Keyword, e.g., 'Elara and Kaelen's bond']", "[Defining Event from their past]", "Protagonist of %s"]). Keywords should include terms a user might type.
      - "insertion_order": Unique number. "priority": Very High (e.g., 95-100). "enabled": true.

2.  SIGNIFICANT SECONDARY CHARACTERS & NOTABLE NPCs (At least 8-12 entries):
    For important supporting characters, mentors, rivals, key quest-givers, faction leaders not covered as primary, or notable recurring figures who contribute to the world's richness:
      - "comment": "Secondary Character: [Character Name] - [Specific Role/Affiliation, e.g., 'Head Enchanter of the Azure Circle', 'Infamous Smuggler of Port Nyx']",
      - "content": "Deeply detailed description: Appearance (distinguishing features, typical attire), personality (key traits, demeanor, quirks, personal beliefs), motivations/goals (even if seemingly minor, explore them), their specific role and significance to the plot or main characters, relevant background and personal history (what made them who they are?), key relationships and allegiances. Even for minor characters, provide enough detail (several rich sentences) to make them memorable and feel integral to their specific context. What unique knowledge, perspective, skills, or secrets do they hold? How do they influence the local environment or narrative, even in small ways?",
      - "keys": JSON array of 5-7 specific keywords (e.g., ["[Character Name]", "[Role/Title]", "[Associated Faction/Location]", "[Key Trait/Skill]", "[Unique Knowledge Area]"]).
      - "insertion_order": Unique. "priority": (e.g., 75-90). "enabled": true.

3.  KEY LOCATIONS (At least 6-9 entries):
    For major cities, distinct regions, significant landmarks, hidden areas, important buildings, unique natural wonders:
      - "comment": "Location: [Location Name] - [Type & Region, e.g., 'The Obsidian Citadel - Volcanic Fortress in the Ash Wastes', 'Whispering Glades - Ancient Elven Forest']",
      - "content": "Exhaustive description using vivid imagery and sensory details: Appearance (architecture, geography, dominant colors, textures), atmosphere (e.g., bustling, desolate, eerie, serene – what contributes to this?), ambient sounds, typical smells, the 'feel' of the place. Detailed history (founding, major events that occurred there, cultural significance, ruins or remnants of past eras). Notable inhabitants/creatures (species, specific NPCs, unique monsters, their behaviors). Strategic or cultural importance. Unique features (magical properties, rare resources, architectural marvels). Flora, fauna, and unique ecological aspects. Legends, myths, local folklore, and even ghost stories associated with the location. What secrets or hidden areas might exist? What does daily life look like for its inhabitants? What are common dangers or points of interest? Current events or ongoing conflicts.",
      - "keys": JSON array of 5-7 keywords (e.g., ["[Location Name]", "[Region]", "[Type of Place]", "[Notable Feature/Landmark]", "[Associated Faction/Event]", "[Dominant Atmosphere/Sensory Detail]", "[Key Resource/Flora/Fauna]"]).
      - "insertion_order": Unique. "priority": (e.g., 70-90). "enabled": true.

4.  MAJOR FACTIONS/ORGANIZATIONS (At least 5-7 entries):
    For influential guilds, kingdoms, empires, cults, corporations, rebel groups, secret societies, knightly orders:
      - "comment": "Faction: [Faction Name] - [Type & Allegiance, e.g., 'The Silver Hand Paladins - Holy Order', 'Nightscale Syndicate - Criminal Cartel']",
      - "content": "Comprehensive information: Goals (stated vs. actual), core ideology and philosophies, detailed structure and hierarchy (ranks, leadership roles, internal politics, potential schisms), profiles of current and past notable leaders (can reference character entries). Key members and their influence. Areas of operation and territories controlled. All resources (military, economic, magical, technological, informational). Influence on local and global politics. Allies and enemies (nature of these relationships: treaties, rivalries, open war, espionage). Propaganda vs. true actions. Public perception vs. internal realities. Recruitment methods and initiation rituals. Symbols, mottos, and heraldry. Significant historical achievements, atrocities, or turning points. Their specific impact on the daily lives of ordinary people within their sphere of influence.",
      - "keys": JSON array of 5-7 keywords (e.g., ["[Faction Name]", "[Leader Name]", "[Faction Type]", "[Base/Territory]", "[Core Ideology Keyword]", "[Symbol/Motto]", "[Key Ally/Enemy]"]).
      - "insertion_order": Unique. "priority": (e.g., 80-95). "enabled": true.

5.  PIVOTAL HISTORICAL EVENTS (At least 4-6 entries):
    For past wars, significant discoveries, cataclysms, founding moments, magical surges, divine interventions, or legendary occurrences that profoundly shaped the current setting:
      - "comment": "Event: [Event Name] - [Era & Impact, e.g., 'The Great Schism - Religious Upheaval, 500 years ago', 'The Starfall Prophecy - Ongoing Cosmic Event']",
      - "content": "Detailed account: Underlying causes and preceding conditions. Key figures, factions, and nations involved. Detailed summary of how the event unfolded (major battles, political maneuvers, social upheavals, discoveries made). Immediate consequences (casualties, territorial changes, treaties, societal shifts). Profound and lasting long-term impacts on the world, cultures, politics, geography, magic, technology, and current state of affairs. Include differing historical interpretations or perspectives on the event if they exist within the fictional world (e.g., victor's history vs. an oppressed group's account). What cultural trauma or triumphs stemmed from this event? Any archaeological evidence, surviving artifacts, songs, or folklore related to it? Were there prophecies before or after related to this event?",
      - "keys": JSON array of 5-7 keywords (e.g., ["[Event Name]", "[Historical Period/Century]", "[Key Figure/Faction in Event]", "[Primary Impact Keyword, e.g., 'Magical Cataclysm']", "[Location of Event]", "[Long-term Consequence]"]).
      - "insertion_order": Unique. "priority": (e.g., 70-90). "enabled": true.

6.  CORE CONCEPTS/WORLD-BUILDING (At least 6-8 entries, covering diverse topics below):
    For foundational elements that define the unique fabric of the series. Aim for truly deep explanations.
      - "comment": "Concept: [Name] - [Specific Category, e.g., 'The Weave - Cosmic Magic System', 'Aethelgardian Pantheon - Major Deities', 'Sunstone Technology - Energy Source', 'The Great Cycle - Reincarnation Belief']",
      - "content": "In-depth explanation. For each concept, delve into:
        *   **Magic Systems:** Sources of power (arcane, divine, elemental, psionic, etc.), casting methods (incantations, runes, gestures, components, innate), rules and limitations (costs, risks, paradoxes, forbidden practices), different schools/traditions/philosophies of magic, famous or infamous practitioners and their unique applications or perversions of magic. Societal impact: Is magic common or rare? Feared or revered? Regulated or wild? Ethical dilemmas posed by its existence.
        *   **Technology/Science:** Level of advancement (steampunk, magitech, medieval, futuristic), key inventions and their inventors, how technology integrates with or conflicts with magic (if present), societal adoption and impact, unintended consequences, scientific theories or paradigms prevalent in the world.
        *   **Cosmology & Deities:** Creation myths in full from various cultures if applicable. Structure of the universe (planes of existence, celestial bodies and their astrological/magical influences). For EACH significant deity: domains, symbols, dogma, detailed worship practices (rituals, prayers, sacrifices, holy days), clergy structure, known divine interventions or periods of silence, relationship with other deities (alliances, rivalries, pantheon structure), how faith manifests in daily life and culture. Schisms or heretical beliefs related to them.
        *   **Flora & Fauna:** Describe 3-5 unique and notable plants AND 3-5 unique animals/monsters in detail. Include their appearance, habitats, behaviors, properties (magical, medicinal, poisonous, edible, crafting materials), and their role in the ecosystem, local folklore, or as symbols.
        *   **Economy & Social Structure:** Currency systems (names of coins, exchange rates if complex), major industries and trade goods, dominant trade routes (land and sea), powerful merchant guilds or corporations. Social classes (nobility, clergy, merchants, peasantry, slaves, outcasts), possibilities for social mobility, inheritance laws, common professions.
        *   **Culture & Daily Life:** Common languages and dialects (perhaps a few sample words or phrases). Dominant art forms, music styles (instruments, famous songs), literature (epic poems, famous authors, playwriting). Mythology and folklore (beyond major historical events). Common sports, games, and leisure activities. Major festivals and holidays (their origins and how they are celebrated). Cuisine (staple foods, delicacies, regional specialties, common drinks). Fashion and typical attire for different classes or regions. Education systems. Marriage customs, family structures, and funerary rites.
        *   **Prophecies or Ancient Curses:** Detail significant, world-impacting prophecies or curses: their full text (if known), origin, interpretations, attempts to fulfill or avert them, and their perceived influence on current events.",
      - "keys": JSON array of 5-7 keywords (e.g., ["[System/Concept Name]", "[Related Terminology]", "[Key Principle/Deity]", "[Limitation/Cultural Impact]", "[Example of Use/Practice]", "[Associated Symbol/Ritual]"]).
      - "insertion_order": Unique. "priority": High (e.g., 85-100). "enabled": true.

For ALL entries:
  - "keys": MUST be a JSON array of appropriately specific, diverse, and numerous string keywords that users might employ to find this information. Think synonyms and related concepts.
  - "content": MUST be exceptionally detailed, descriptive, evocative, and informative, often spanning multiple rich paragraphs. Provide concrete examples, and "show, don't just tell." Explore nuances and avoid simplistic explanations.
  - "insertion_order": A unique integer. Plan thoughtfully for logical flow or priority.
  - "enabled": true (boolean).
  - "priority": An optional integer (e.g., 0-100). Assign thoughtfully based on importance.
  - "comment": A brief, descriptive comment about the entry's topic for easier management.

The entire output MUST be a single, complete, and valid JSON object. Leave no aspect of '%s' unexplored.
`, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series)

		// Call Gemini API
		aiResponse, err := callGeminiAI(ctx, model, prompt)
		if err != nil {
			sendJSONError(w, fmt.Sprintf("AI generation failed for Comprehensive Lorebook: %v", err), http.StatusInternalServerError, logIdentifier)
			return
		}

		// Attempt to unmarshal the AI response into Lorebook struct
		var loreBook Lorebook
		if err := json.Unmarshal([]byte(aiResponse), &loreBook); err != nil {
			log.Printf("Failed to unmarshal Comprehensive Lorebook: %v. AI Response (check logs for ID %s): %s", err, logIdentifier, aiResponse)
			sendJSONError(w, fmt.Sprintf("Failed to parse AI response for Comprehensive Lorebook. Raw AI output (check logs for ID %s for details): %s", logIdentifier, aiResponse[:min(600, len(aiResponse))]), http.StatusInternalServerError, logIdentifier)
			return
		}
		// Ensure root lorebook and all entries are enabled
		loreBook.Enabled = true
		if loreBook.Name == "" {
			loreBook.Name = fmt.Sprintf("Comprehensive Lore for %s", requestPayload.Series)
		} // Default name if AI misses it
		for i := range loreBook.Entries {
			loreBook.Entries[i].Enabled = true
		}

		// Marshal back to indented JSON for saving and response
		jsonData, _ := json.MarshalIndent(loreBook, "", "  ")
		generatedJSONString = string(jsonData)
		// Save the generated JSON to a file
		filePath, err := saveJSONToFile(requestPayload.Series, "lorebook_comprehensive", loreBook.Name, logIdentifier, jsonData)
		if err != nil {
			finalMessage += fmt.Sprintf("Successfully generated Comprehensive Lorebook JSON, but FAILED to save to server file system. Error: %s\n", err.Error())
			log.Printf("Failed to save Comprehensive Lorebook JSON to file: %v", err)
		} else {
			finalMessage += fmt.Sprintf("Successfully generated Comprehensive Lorebook JSON and saved to: %s\n", filePath)
		}
		finalMessage += "Comprehensive Lorebook generation complete.\n"

	case "2": // Lorebook + Narrator Character card - REFINED MULTI-STEP
		optionText = "Narrator Card + Master Lorebook (Refined)"
		finalMessage = fmt.Sprintf("Processing Option 2: Narrator Card + Master Lorebook for '%s'. This is a multi-step process.\n\n", requestPayload.Series)
		var allGeneratedJSONsOpt2 []string // To hold JSON strings of multiple generated artifacts

		// --- Step 1: Generate Narrator Character Card ---
		finalMessage += "Step 1: Generating highly detailed Narrator Character Card...\n"
		narratorName := fmt.Sprintf("The Narrator of %s", requestPayload.Series) // Standardized Narrator name

		// Refined prompt for the Narrator Character Card as a storytelling guideline framework
		narratorCardPrompt := fmt.Sprintf(`
Generate an exceptionally detailed and comprehensive SillyTavern V2 Character Card JSON for a STORYTELLING FRAMEWORK called the Narrator Framework for the series '%s'.
This is NOT an actual character in the story, but rather a meta-entity that provides guidelines, principles, and frameworks for storytelling, character interpretation, and narrative techniques specific to the '%s' series.
Your ENTIRE response MUST be ONLY a single, valid JSON object, starting with '{' and ending with '}'. No other text, comments, explanations, or markdown formatting should precede or follow this JSON object.
The JSON object must strictly adhere to the SillyTavern V2 Character Card specification:
  "spec": "chara_card_v2",
  "spec_version": "2.0".

The "data" object must include ALL the following fields, filled with rich, exceptionally detailed, and creatively profound content:
  "name": "%s",
  "description": "An extensive, detailed guide to the narrative framework for the '%s' series. Explain the primary storytelling approach that best suits this world (e.g., 'Stories in this world should be told through a balanced mixture of character-driven emotional arcs and plot-driven conflicts, with particular emphasis on the themes of redemption and the cost of power'). Detail the key narrative techniques that work best (e.g., 'Effective storytelling in this world balances foreshadowing with surprising but inevitable revelations, uses environment descriptions to reflect character emotions, and weaves subtle connections between seemingly unrelated elements'). This field MUST embed significant and specific examples of key storytelling principles unique to '%s' (e.g., 'Character arcs should reflect the world's central philosophy that power always exacts a cost, as exemplified by the narrative structure of the Blood Pact storyline where each boon granted requires increasing sacrifice'). Include guidance on how to interpret and portray characters from the lorebook, how to balance different narrative elements, and approaches to create tension and resolution in a way that honors the world's established tone. Example: 'This framework provides comprehensive guidelines for crafting stories within the shadowed realms of Aethelgard – a world forever scarred by the Starfall. Narratives should emphasize personal transformation against a backdrop of ancient mysteries, with particular attention to how characters respond to lost knowledge and forbidden power. Characters should be portrayed with complex motivations, where even virtuous goals often lead to morally ambiguous choices. Dialog should reflect cultural background, with Veridium nobles speaking in formal, layered language while Outland traders use more direct, metaphor-rich expressions. Environmental descriptions should serve as both worldbuilding and emotional mirrors, with weather patterns and architectural details reinforcing the psychological state of viewpoint characters.'",
  "personality": "Detail the storytelling personality and approach that best suits this world. This is not about the narrator as a character, but about the ideal narrative voice and philosophy for telling stories in this setting. Include approaches to pacing, revealing information, creating and resolving tension, and maintaining consistency with the world's established tone. Example: 'Stories in this setting benefit from a measured pace that allows for immersion in sensory details and character introspection, punctuated by moments of sudden action or revelation. Information should be revealed primarily through character experience rather than exposition, with secrets unfolding gradually as characters discover them. Tension derives most effectively from moral dilemmas and conflicting loyalties rather than external threats alone. The narrative voice should maintain a sense of historical context, occasionally zooming out to connect current events to the broader tapestry of the world's history. Dialog should be used to highlight cultural differences and personal philosophies, with attention to how different factions within %s would express similar ideas differently.'",
  "scenario": "Establish the framework for approaching storytelling scenarios in this world. Detail guidance on how to structure scenes, manage transitions between different story elements, and effectively use the specific narrative devices most appropriate for this setting. Example: 'When crafting narratives in '%s', begin by establishing a clear thematic focus that resonates with the world's core conflicts (e.g., tradition vs. progress, order vs. chaos, personal freedom vs. collective responsibility). Structure scenes to reflect the world's natural rhythm - from the frantic pace of urban centers to the contemplative atmosphere of ancient ruins. Transitions between locations should acknowledge travel methods consistent with the world's technology and magic, using these journeys as opportunities for character development. When introducing lore elements from the associated lorebook, present them through the lens of character perception rather than objective truth, allowing for misunderstandings and cultural biases to color the narrative.'",
  "first_mes": "A comprehensive guide for how to begin stories in this world, including advice on establishing setting, introducing characters, and setting initial stakes in a way consistent with the series' style. Include guidance on using the '/Option x' mechanic as a storytelling tool rather than character dialog. Example: 'Welcome, Storyteller. This framework will guide your creation and interpretation of narratives within the world of '%s'. To craft compelling beginnings in this setting, consider these principles: 1) Establish the specific era and regional context immediately, as the cultural and political landscape varies dramatically across both space and time. 2) Introduce characters through meaningful action that reveals both capability and flaw. 3) Reflect the world's essence through sensory details - the metallic scent of thaumaturgy, the distant hum of ancient mechanisms, or the particular quality of light in enchanted forests. \nWhen planning your narrative direction, consider these pathways: \n/Option 1: Character-driven stories focusing on personal growth against the backdrop of larger conflicts. \n/Option 2: Mystery narratives that gradually reveal hidden connections to the world's ancient history. \n/Option 3: Political intrigue centered on faction dynamics and competing philosophies. \nThese approaches can be combined or used as starting points for your unique narrative vision.'",
  "mes_example": "Provide THREE detailed examples of effective storytelling approaches for different scenarios. Each should demonstrate how to use the narrative framework and '/Option x' mechanic to guide story development rather than as character dialog. Separate with \"<START>\".
    Example 1:
    <START>
    {{user}}: I'd like guidance on developing a story about a character discovering forgotten magic.
    {{char}}: For a narrative centered on magical rediscovery in '%s', consider these essential elements:

    First, establish your character's relationship to the established magical traditions - are they an academy dropout, a skeptical scholar, or someone from a culture where magic is viewed differently? This positioning creates the lens through which the discovery will be interpreted.

    The forgotten magic should reflect one of the world's core themes. In '%s', lost magic often carries dual nature: tremendous power paired with unforeseen consequences, reflecting the world's theme of knowledge as a double-edged sword.

    Structure your narrative arc with these phases:
    1. Initial discovery (small, intriguing manifestation)
    2. Experimental application (growing power, subtle warnings)
    3. Complication (the magic's true nature or cost revealed)
    4. Integration or rejection (character's moral choice about this power)

    Consider these specific development paths:
    /Option A: The magic is tied to an ancient pact with entities from beyond the Veil - focus on cosmic horror elements and moral ambiguity.
    /Option B: The magic represents a culturally suppressed tradition - explore themes of historical revisionism and cultural reclamation.
    /Option C: The magic requires a fundamental shift in worldview to fully utilize - emphasize personal transformation and philosophical awakening.
    
    Which direction best aligns with your narrative goals?

    Example 2:
    <START>
    {{user}}: How should I approach writing political conflict between factions in this world?
    {{char}}: Political narratives in '%s' are most compelling when they reflect the unique value systems and historical contexts of competing factions. Consider these framework guidelines:

    Unlike simple power struggles, factional conflicts in this world should center on fundamentally different philosophies about how society should function. For example, the historical tension between the Mercantile League and the Old Houses isn't merely about wealth, but about whether prosperity should be inherited or earned.

    Character motivations in political narratives should operate on three levels:
    • Personal ambition or survival
    • Factional loyalty or advancement
    • Genuine ideological belief (which may conflict with the other motivations)

    Political dialog should reflect specific cultural speech patterns - formal diplomatic language masks true intentions, with cultural idioms and metaphors revealing deeper meanings to attentive listeners.

    Consider these political narrative approaches:
    /Option P1: Bottom-up perspective - follow characters of lower status witnessing political machinations from below, emphasizing how high-level decisions impact ordinary lives.
    /Option P2: Multiple viewpoints across faction lines - develop parallel narratives showing how the same events are interpreted differently based on cultural context.
    /Option P3: Focus on a mediator figure with divided loyalties - explore the personal cost of attempting to bridge irreconcilable worldviews.
    
    Each approach provides different insights into the complex political landscape of '%s'.

    Example 3:
    <START>
    {{user}}: I need guidance on integrating supernatural elements from the lorebook into my story.
    {{char}}: When incorporating supernatural elements from the '%s' lorebook, consider these storytelling principles:

    The supernatural should follow consistent internal logic while maintaining an element of mystery. Even in scenes of explicit magical manifestation, leave aspects unexplained to preserve the sense of wonder central to this world's atmosphere.

    Supernatural encounters are most effective when they reveal character. When a character faces the unknown, their reaction should illuminate their values, fears, and adaptability. A scholar might approach a spectral apparition analytically, while a frontier settler might interpret the same entity through folk beliefs.

    The scale of supernatural intervention should match your narrative scope - personal stories might involve subtle hauntings or minor magical boons, while epic narratives can incorporate divine intervention or world-altering magical catastrophes.

    Consider these approaches for supernatural integration:
    /Option S1: Gradual normalization - introduce supernatural elements as initially terrifying or wondrous, then show how characters come to understand and adapt to them, reflecting the world's theme of the unknown becoming known.
    /Option S2: Cultural interpretation - present supernatural phenomena through multiple cultural lenses, with different traditions in '%s' offering conflicting explanations for the same magical events.
    /Option S3: Supernatural as metaphor - align magical manifestations with character psychological states, using the supernatural as an external reflection of internal conflicts.
    
    Remember that in '%s', the supernatural is neither completely chaotic nor fully understood - effective stories maintain this tension between order and mystery.",
  "creator_notes": "This is NOT a character but a storytelling framework providing comprehensive guidelines on narrative techniques, character interpretation, and story structure appropriate for the '%s' series. It contains extensive knowledge about the world's narrative style, thematic elements, and storytelling approaches, but should NOT be presented as an actual character within the fiction. Its primary purpose is to serve as a meta-level guide for crafting stories, developing characters, and maintaining consistent tone within this world.",
  "system_prompt": "You are {{char}}, a comprehensive STORYTELLING FRAMEWORK for the '%s' series - not an actual character in the narrative. Your purpose is to provide detailed guidance on effective storytelling techniques, character interpretation principles, and narrative approaches specific to this fictional world. When responding to questions, offer concrete storytelling advice, narrative structure suggestions, and guidance on interpreting elements from the lorebook. Focus on HOW to tell stories in this world rather than telling the stories yourself. Use the '/Option x' format to suggest different narrative approaches, character development paths, or thematic explorations relevant to the user's question. Always maintain your role as a meta-level storytelling guide rather than a character within the fiction. Your advice should help writers and roleplayers craft narratives that feel authentic to the established tone, themes, and world logic of '%s'.",
  "post_history_instructions": "Continue to provide specific, actionable storytelling guidance rather than speaking as a character in the narrative. Build upon previous discussions to offer increasingly tailored advice. When the user selects an '/Option x' path, provide more detailed guidance for that specific narrative approach. Maintain consistency in your storytelling principles while adapting to the user's evolving needs and questions. Remember that your purpose is to help the user develop their own stories within the '%s' universe, not to tell stories to them.",
  "alternate_greetings": [
    "Welcome to the narrative framework for '%s'. This guide offers comprehensive principles for storytelling in this unique world, with particular attention to its distinctive themes, character archetypes, and narrative rhythms. When developing your stories, consider these structural approaches: \n/Option 1: 'The Hero's Descent' - Focus on a protagonist who gains power at significant personal cost, reflecting the world's theme of sacrifice and consequences. \n/Option 2: 'The Converging Paths' - Develop multiple character storylines that initially seem separate but gradually reveal connections to a central mystery or conflict. \n/Option 3: 'The Cultural Lens' - Explore familiar tropes through the unique cultural perspectives of different factions in this world.",
    "This storytelling framework will help you craft authentic narratives within the '%s' universe. Remember that the most compelling stories in this setting tend to balance three key elements: 1) Personal character journeys that reflect growth or transformation, 2) Exploration of the world's unique systems and cultures, and 3) Thematic resonance with the setting's core philosophical questions. Consider these narrative foundations: \n/Option A: Begin with a character confronting a belief or tradition they've long accepted without question. \n/Option B: Start with a mystery whose solution reveals unexpected connections between disparate elements of the world. \n/Option C: Focus on a conflict between competing valid viewpoints, each with cultural and historical justification."
    ],
  "tags": ["%s", "Storytelling Framework", "Narrative Guide", "Worldbuilding", "Character Development", "Plot Structure", "Thematic Analysis", "AI Generated", "SillyTavern V2"],
  "creator": "AI Fiction Forge (Narrator Framework v1.0)",
  "character_version": "1.0F",
  "visual_description": "This is a meta-level storytelling framework, not a character with a physical appearance. If visualization is needed for interface purposes, it could be represented as an elegant leather-bound book with the title '%s Storytelling Guide' embossed in gold, opened to reveal pages filled with narrative diagrams, character interpretation guidelines, and world-specific storytelling principles. The pages might appear to gently glow with subtle illumination, highlighting key sections relevant to the current discussion.",
  "thought_pattern": "This framework organizes storytelling principles along thematic, structural, and cultural dimensions. It categorizes narrative techniques by their effectiveness in different scenarios within '%s', considering how various character types, plot structures, and thematic elements can be authentically developed within this world's established logic and atmosphere. When providing guidance, it considers the user's specific storytelling goals, matches them with appropriate techniques from the framework, and offers structured pathways for narrative development.",
  "speech_pattern": "This framework communicates in clear, instructional language focused on storytelling methodology. It uses precise literary terminology when helpful but always explains concepts in accessible terms. It frames advice as concrete suggestions rather than abstract theory, offering specific examples from the world of '%s' to illustrate key points. It uses organizational elements like numbered lists, categorized options, and clearly delineated alternative approaches to help users navigate complex narrative decisions."

Do NOT include any text, comments, or markdown formatting outside the main, single JSON object.
The entire response MUST be a single, complete, and valid JSON object.
`, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series)

		// Call Gemini API for Narrator Card
		narratorCardAIResponse, err := callGeminiAI(ctx, model, narratorCardPrompt)
		if err != nil {
			finalMessage += fmt.Sprintf("  ERROR generating Narrator Card: %v\n", err)
			// Do not proceed if Narrator card fails, as it's crucial for this option.
			sendJSONError(w, fmt.Sprintf("AI generation failed for Narrator Card (Option 2, Step 1): %v", err), http.StatusInternalServerError, logIdentifier)
			return
		}

		var narratorCard CharacterCardV2
		if err := json.Unmarshal([]byte(narratorCardAIResponse), &narratorCard); err != nil {
			log.Printf("Failed to unmarshal Narrator Card: %v. AI Response (check logs for ID %s): %s", err, logIdentifier, narratorCardAIResponse)
			finalMessage += fmt.Sprintf("  ERROR parsing Narrator Card. Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, narratorCardAIResponse[:min(600, len(narratorCardAIResponse))])
			sendJSONError(w, fmt.Sprintf("Failed to parse AI response for Narrator Card (Option 2, Step 1). Raw AI output (check logs for ID %s for details): %s", logIdentifier, narratorCardAIResponse[:min(600, len(narratorCardAIResponse))]), http.StatusInternalServerError, logIdentifier)
			return
		}
		// Ensure critical fields are set for Narrator Card
		if narratorCard.Spec == "" {
			narratorCard.Spec = "chara_card_v2"
		}
		if narratorCard.SpecVersion == "" {
			narratorCard.SpecVersion = "2.0"
		}
		if narratorCard.Data.Name == "" {
			narratorCard.Data.Name = narratorName
		} // Default name
		narratorCard.Data.CharacterBook = nil // Explicitly ensure no embedded lorebook for Narrator

		// Marshal Narrator card to JSON and save
		narratorJsonData, _ := json.MarshalIndent(narratorCard, "", "  ")
		allGeneratedJSONsOpt2 = append(allGeneratedJSONsOpt2, string(narratorJsonData)) // Add to list for final response
		narratorFilePath, saveErr := saveJSONToFile(requestPayload.Series, "narrator_card", narratorCard.Data.Name, logIdentifier, narratorJsonData)
		if saveErr != nil {
			finalMessage += fmt.Sprintf("  Successfully generated Narrator Card, but FAILED to save. Error: %s\n", saveErr.Error())
		} else {
			finalMessage += fmt.Sprintf("  Successfully generated and saved Narrator Card to: %s\n", narratorFilePath)
		}
		finalMessage += "Step 1: Narrator Character Card generation complete.\n\n"

		// --- Step 2: Generate Master Lorebook ---
		finalMessage += "Step 2: Generating Master Lorebook (most complete, 50-60+ entries). This may take some time...\n"
		masterLorebookName := fmt.Sprintf("Master Lorebook for %s", requestPayload.Series)

		// Refined prompt for the Master Lorebook, emphasizing its role as the Narrator's knowledge base
		masterLorebookPrompt := fmt.Sprintf(`
Generate the ABSOLUTELY MOST COMPLETE, EXHAUSTIVE, AND DEEPLY DETAILED SillyTavern V2 Lorebook JSON possible for the series '%s'.
This Master Lorebook must be the ultimate, unparalleled repository of all knowledge for this universe, aiming for no practical limit on entries to cover every conceivable aspect, including obscure lore, hidden histories, and subtle nuances. It should contain a vast ocean of interconnected information, serving as the definitive canonical knowledge base for an omniscient Narrator of '%s'. Spare absolutely no detail; delve into micro-details and intricacies. If it could exist or be known within this world, document it here.
Your ENTIRE response MUST be ONLY a single, valid JSON object, starting with '{' and ending with '}'. No other text, comments, explanations, or markdown formatting should precede or follow this JSON object.
The JSON object must strictly adhere to the SillyTavern V2 Lorebook specification.

Lorebook Root Structure:
  \"name\": \"%s - The Definitive Canon\",
  \"description\": \"The ultimate, definitive, and most comprehensive collection of lore for the world of \'%s\'. This Master Lorebook delves into extreme, exhaustive detail on all major, minor, and even background characters; every significant, minor, and rumored location; pivotal and obscure historical events from creation myths to yesterday\'s whispers; all known and suspected factions and organizations; intricate and subtle relationships; core and esoteric world-building elements (magic systems, technologies, cosmologies, deities, cultures, species, prophecies, economies, social structures, languages, flora, fauna); and any other piece of relevant information that defines this series. This is the ultimate knowledge base for the Narrator, designed for unparalleled depth and exploration. Content within entries should use Markdown for clarity (lists, bolding for key terms, italics for emphasis or in-world quotes) where it enhances readability without sacrificing detail. Complex data within an entry (like a list of planetary systems in a sci-fi setting, or noble house lineages) can be structured using Markdown lists or simple textual tables if appropriate.\",
  \"scan_depth\": 50,  // Maximize scan depth for broad context matching and subtle triggers.
  \"token_budget\": 8000, // Very generous token budget for exceptionally rich entries.
  "insertion_order": 0,
  "enabled": true,
  "recursive_scanning": true, // Essential for deep contextual connections and emergent lore discovery.

It MUST contain AT LEAST 60-75+ (aim for 75 or more if the series' depth allows; the more entries and the more detail in each, the better – strive for absolute, exhaustive completeness) distinct and EXCEPTIONALLY, PROFOUNDLY, and INTRICATELY detailed 'entries'. Strive for maximum comprehensiveness across diverse categories. Explicitly state or hint at interconnections between entries to weave a cohesive world.

1.  MAJOR CHARACTERS (All significant protagonists, antagonists, and pivotal figures; aim for 10-15+ entries):
    Provide an unparalleled depth of information, far beyond a simple summary.
      - "comment": "Major Character: [Character Name] - [Their Full Title and Primary Role in the Grand Narrative, e.g., 'Valerius Kael, the Last Dragonlord, Bearer of the Sundered Crown']",
      - "content": "EXHAUSTIVELY detailed profile:
        *   **Appearance:** Vivid, multi-sensory descriptions (visual specifics, attire for various occasions, unique features, common expressions, voice timbre, scent if notable, how they carry themselves).
        *   **Personality:** Complex psychological profile (nuances, contradictions, internal conflicts, core philosophies, virtues, vices, fears, deepest desires, how they handle stress/joy/loss, character development arcs and their catalysts). Include defining quotes or eloquent phrases attributed to them.
        *   **History & Lineage:** Extensive background (birth circumstances, lineage tracing back generations if significant, key life events from childhood to present, formative experiences, turning points, greatest triumphs and most devastating failures).
        *   **Motivations & Goals:** Intricate, evolving motivations and both short-term and long-term goals. What drives them at their core? What are their hidden agendas?
        *   **Impact & Relationships:** Profound impact on the series' narrative, other characters, and the world itself. Detailed analysis of ALL key relationships (alliances, rivalries, romantic ties, family dynamics, mentorships, dependents, pets/familiars) – analyze their nature, history, power dynamics, emotional core, and how these relationships shape the character and are shaped by them. How are they perceived by different factions or cultures?
        *   **Abilities & Possessions:** Signature abilities, skills, talents (mundane and magical/supernatural), knowledge domains. Detailed descriptions of notable possessions (weapons, armor, artifacts, tools, residences, modes of transport), including their history, powers, and significance.
        *   **Strengths & Weaknesses:** Both overt and subtle strengths and weaknesses (physical, mental, emotional, magical). How do they leverage strengths and mitigate or succumb to weaknesses?
        *   **Speech Patterns & Mannerisms:** Describe their typical way of speaking, common phrases, verbal tics, and characteristic mannerisms or habits.",
      - "keys": JSON array of 8-12 highly specific and diverse keywords (e.g., ["[Full Name]", "[All Known Aliases]", "[Specific Title/Role, e.g., 'Dragonlord']", "[Key Relationship Pair, e.g., 'Valerius and Lyra's Vow']", "[Defining Personal Tragedy/Triumph]", "[Signature Weapon/Artifact Name]", "[Core Philosophical Belief]", "[Psychological Trait, e.g., 'Valerius's Survivor Guilt']", "Major Character %s", "[Associated Faction/Homeland]", "[Unique Ability Keyword]"]).
      - "insertion_order": Unique. "priority": Extremely High (e.g., 100). "enabled": true.

2.  SIGNIFICANT SECONDARY & MINOR CHARACTERS (All other named characters who have any role, however small, including recurring NPCs, quest givers, shopkeepers with personality, local figures, historical mentions, etc.; aim for 25-35+ entries):
      - "comment": "Supporting Character: [Character Name] - [Their Specific Function/Occupation and Location, e.g., 'Old Man Hemlock, the Hermit Apothecary of Greywood']",
      - "content": "Deeply detailed information:
        *   **Appearance:** Distinguishing features, typical clothing, any notable items they carry.
        *   **Personality:** Key traits, demeanor, quirks, personal beliefs, fears, and desires (even if simple).
        *   **History:** Personal history relevant to their role, interactions, and knowledge. What brought them to their current situation?
        *   **Motivations & Role:** Their purpose in the narrative, even if it's just to provide a piece of information or a specific service. Relationship to major characters or events.
        *   **Knowledge/Skills:** Any unique skills, specialized knowledge (e.g., local history, gossip, crafting recipe), rumors they might spread or be privy to. Even minor characters should have several rich sentences to make them feel like a living part of the world, not just a plot device. What lesser-known facts or perspectives might they offer?",
      - "keys": JSON array of 5-8 keywords (e.g., ["[Character Name]", "[Occupation/Role]", "[Location they frequent]", "[Associated Quest/Item/Information they provide]", "[Key Characteristic/Quirk]", "[Relationship to a Major Character/Faction if any]"]).
      - "insertion_order": Unique. "priority": (e.g., 70-90). "enabled": true.

3.  EVERY POSSIBLE LOCATION (Major cities, towns, villages, distinct regions, specific landmarks, dungeons, ruins, natural wonders, cosmic locations, important buildings like castles, temples, inns, shops, individual houses if significant, etc.; aim for 25-35+ entries):
      - "comment": "Location: [Location Name] - [Detailed Type/Region/Significance, e.g., 'The Sunken Library of Azmar - Ancient Archive, Abyssal Depths']",
      - "content": "Exhaustive, multi-sensory description:
        *   **Geography & Layout:** Detailed geography/architecture/layout (include textual descriptions of maps if possible, e.g., 'The city is built on seven hills...'). Dominant architectural styles and materials.
        *   **Atmosphere & Sensory Details:** Prevailing atmosphere (e.g., bustling, oppressive, sacred, decaying) and the sensory details that create it (specific sights, ambient sounds, common smells, tactile sensations like temperature or humidity, even tastes if relevant like salty air or metallic tang).
        *   **History & Legends:** Complete history (founding, key events that occurred there, cultural significance, different eras of occupation/control). Legends, myths, ghost stories, and local folklore associated with the place. Archaeological discoveries or ruins.
        *   **Inhabitants & Ecology:** All notable inhabitants (species, specific NPCs, monsters, spirits). Unique flora, fauna, and ecological features or anomalies. How do inhabitants interact with their environment?
        *   **Significance & Resources:** Strategic, cultural, magical, economic, or religious significance. Available resources, goods, or unique products.
        *   **Secrets & Points of Interest:** Hidden secrets, concealed areas, dungeons, traps, puzzles, or points of interest for explorers or scholars.
        *   **Daily Life & Culture:** Ongoing events, conflicts, political status, economic activities, typical weather patterns, local customs, dialects, or traditions specific to the location. What is daily life like for various social strata within this location?",
      - "keys": JSON array of 6-10 keywords (e.g., ["[Location Name]", "[Specific District/Sub-Area if any]", "[Region it's in]", "[Type of Place, e.g., Ancient Ruin, Capital City]", "[Notable Architectural Feature/Landmark]", "[Dominant Sensory Detail/Atmosphere Keyword]", "[Key Historical Event that occurred here]", "[Associated Faction/Character]", "[Unique Resource/Flora/Fauna]", "[Local Legend/Secret Keyword]"]).
      - "insertion_order": Unique. "priority": (e.g., 80-95). "enabled": true.

4.  ALL FACTIONS & ORGANIZATIONS (Kingdoms, empires, republics, city-states, guilds of all types, cults, secret societies, criminal enterprises, trading companies, rebel groups, knightly orders, magical circles, philosophical schools, etc.; aim for 10-15+ entries):
      - "comment": "Faction: [Faction Name] - [Detailed Type & Primary Goal/Ideology, e.g., 'The Obsidian Hand - Shadowy Assassin Cult dedicated to Cosmic Balance']",
      - "content": "Comprehensive, in-depth details:
        *   **History & Origins:** Full history (founding myths/facts, major turning points, periods of growth/decline, significant past leaders and their legacies).
        *   **Ideology & Goals:** Complete ideology, philosophies, religious tenets (if any), stated public goals vs. secret or true agendas. What are their core values and taboos?
        *   **Structure & Hierarchy:** Detailed organizational structure, internal hierarchy (specific ranks, titles, roles, responsibilities), leadership councils or figures, internal politics, factions within the faction, and potential schisms.
        *   **Membership & Recruitment:** Profiles of current and past notable leaders and key members (can reference character entries). Recruitment methods, initiation rituals, criteria for membership, benefits and drawbacks of joining. Public perception and reputation.
        *   **Operations & Influence:** Primary areas of operation, territories controlled or influenced, spheres of interest (political, economic, magical, etc.). Methods of exerting influence.
        *   **Resources & Assets:** All resources (military strength, economic power, magical capabilities, technological advantages, information networks, ancient artifacts, unique knowledge).
        *   **Relationships:** Intricate relationships with other factions and notable individuals (alliances, rivalries, open wars, cold wars, trade agreements, espionage efforts, betrayals, blood feuds).
        *   **Symbols & Culture:** Symbols, heraldry, mottos, secret codes or languages, internal culture and traditions.
        *   **Achievements & Atrocities:** Significant past achievements, discoveries, or, conversely, infamous atrocities or failures.",
      - "keys": JSON array of 6-10 keywords (e.g., ["[Faction Name]", "[Current Leader of Faction]", "[Faction Type, e.g., Mage Guild, Kingdom]", "[Primary Base/Territory]", "[Core Ideology Keyword, e.g., 'Purification', 'Knowledge Hoarding']", "[Notable Member/Symbol]", "[Key Ally/Enemy Faction]", "[Secret Goal Keyword]", "[Recruitment Method/Ritual]"]).
      - "insertion_order": Unique. "priority": (e.g., 85-100). "enabled": true.

5.  MAJOR & MINOR HISTORICAL EVENTS (All shaping events from ancient mythology and creation stories to recent past events that impact the current setting, including those known only through obscure texts or oral traditions; aim for 15-20+ entries):
      - "comment": "Event: [Event Name] - [Era/Date if known & Brief Nature, e.g., 'The Night of Weeping Stars - Cosmic Anomaly, Elder Age']",
      - "content": "Thorough, multi-faceted account:
        *   **Context & Causes:** Underlying causes, preceding conditions, prophecies or omens related to it.
        *   **Participants:** All key figures, factions, nations, species, or even deities involved. Who were the instigators, victims, heroes, villains?
        *   **Unfolding:** Detailed unfolding of the event (major battles, political maneuvers, discoveries, social upheavals, magical phenomena, divine interventions). Include specific dates, locations, and turning points.
        *   **Consequences:** Immediate consequences (casualties, territorial changes, treaties, destruction/creation of artifacts or landmarks).
        *   **Long-Term Impact:** Profound and lasting long-term impacts on the world's societies, cultures, politics, geography, magic, technology, environment, inter-species relations, and the current state of affairs. How is this event remembered or commemorated (or suppressed)?
        *   **Perspectives & Interpretations:** Differing historical interpretations, myths, legends, or lost knowledge surrounding the event. Are there revisionist histories or contested truths? Include primary source snippets if they can be invented (e.g., 'A fragment from a soldier's diary reads...').
        *   **Legacy:** The event's depiction in art, song, literature, or folklore. Any surviving artifacts, ruins, or living memories connected to it.",
      - "keys": JSON array of 6-10 keywords (e.g., ["[Event Name]", "[Specific Historical Period/Century, e.g., 'Third Dragon War']", "[Key Figure Central to Event]", "[Primary Impact of Event, e.g., 'Fall of Eldoria Empire']", "[Primary Location of Event]", "[Faction Most Affected/Involved]", "[Key Consequence/Legacy Keyword]", "[Associated Prophecy/Artifact]"]).
      - "insertion_order": Unique. "priority": (e.g., 80-95). "enabled": true.

6.  OVERALL LORE & WORLD-BUILDING (This section MUST be vast, covering every conceivable element that defines the world. Aim for 20-30+ entries, each deeply exploring a specific facet):
      - "comment": "Lore/Concept: [Specific Name] - [Detailed Category, e.g., 'The Etherium - Source of All Magic', 'The Pantheon of Ashai - Gods of Creation & Destruction', 'Kharidian Steel - Unique Alloy Properties', 'The Great Migration - Racial Origin Story']",
      - "content": "Exhaustive, multi-paragraph explanation for each concept. Leave no aspect unexplored.
        *   **Cosmology & Planes:** Detailed maps/descriptions of planes of existence (material, ethereal, astral, elemental, heavens, hells, dreamscapes, etc.), celestial bodies (suns, moons, planets, constellations) and their influences (astrological, magical, tidal), creation myths from ALL major cultures within the world, structure of the cosmos, known portals or methods of interplanar travel.
        *   **Deities/Pantheons/Religions:** For EACH deity or powerful spiritual entity: domain, symbols, titles, detailed dogma and tenets, complete mythology (birth, deeds, relationships, death/rebirth), specific worship practices (prayers, rituals, sacrifices, festivals, holy days), clergy structure and hierarchy, lay worshippers, schisms/heresies/cults, holy sites/relics, known divine interventions or periods of silence, relationship with other deities (alliances, rivalries, familial ties, wars), how faith (or lack thereof) manifests in daily life, ethics, and culture for their followers.
        *   **Magic System(s):** Exhaustive details on ALL sources of power (arcane, divine, elemental, psionic, primal, shadow, etc.), methods of casting/channeling (incantations, runes, gestures, foci, components, innate talent, pacts), strict rules and limitations (costs, risks, backlash, paradoxes, forbidden practices, societal taboos), different schools/traditions/philosophies of magic and their interrelations, famous or infamous practitioners and their unique applications or perversions of magic. Societal impact: Is magic common or rare? Feared or revered? Regulated or wild? Who can use it? How is it taught? Ethical dilemmas posed by its existence. Magical creatures, their nature, and their connection to magic. Creation and properties of magical artifacts. Interactions between different magic systems.
        *   **Species & Races (Intelligent & Monstrous):** For EACH distinct species/race (humanoid, beastly, elemental, undead, construct, etc.): detailed physiology (appearance, senses, lifespan, reproduction, diet, vulnerabilities, unique abilities), typical psychological traits and tendencies, complex societal structure (family units, governance, laws, social castes), rich culture (art, music, literature, oral traditions, customs, values, ethics, fashion, cuisine), detailed history and origin myths, inter-species relations (alliances, prejudices, wars, trade, integration), notable individuals or heroes/villains of that species. For monsters: habitat, behavior, attack forms, weaknesses, ecological role, lore/myths about them.
        *   **Flora & Fauna (Unique & Mundane):** Describe numerous unique and notable plants, animals, fungi, and other lifeforms. For each: detailed appearance, habitat, behaviors, properties (magical, medicinal, poisonous, edible, crafting materials, symbolic meaning), and their role in the ecosystem, local folklore, agriculture, or as symbols. Include mundane creatures if they play a significant role.
        *   **Economy & Trade:** Currencies (names of coins, materials, exchange rates, debasement issues), banking systems (usury, letters of credit), major industries (agriculture, mining, crafting, fishing, etc.), key resources and who controls them, detailed trade routes (land and sea, dangers, major trading posts), powerful merchant guilds or corporations and their influence, black markets and illicit trade, taxation systems, economic theories or policies.
        *   **Politics & Governance:** Common types of government (monarchy, republic, aurocracy, theocracy, tribal, etc.) and specific examples. Detailed legal systems (codes of law, courts, trials, punishments), political factions (beyond major ones, e.g., courtly cliques, reform movements), succession laws, systems of nobility, diplomacy and treaties, methods of warfare and military structures, espionage networks, and civil services.
        *   **Social Structure & Daily Life:** Class systems (nobility, clergy, merchants, artisans, peasantry, slaves, outcasts) and their interrelations, possibilities for social mobility, family structures and kinship systems, gender roles and expectations (and exceptions), education systems (access, curriculum, institutions), common professions and crafts, healthcare and healing practices (magical and mundane), sanitation, housing types, daily routines for different social classes.
        *   **Culture & Arts:** Detailed descriptions of languages (alphabets/scripts if describable, grammar nuances, key phrases, dialects, pidgins, sign languages, dead languages studied by scholars). Major art forms (painting, sculpture, music, dance, theatre, literature, oral storytelling) and their styles, famous artists/works. Musical instruments and traditions. Mythology, folklore, epic poems, famous proverbs and sayings. Common sports, games, and leisure activities. Major festivals, holidays, and celebrations (their origins, rituals, and how they are observed by different cultures/classes). Cuisine (staple foods, delicacies, regional specialties, common drinks, cooking methods, meal etiquette). Fashion trends and typical attire for different classes, regions, or professions.
        *   **Technology & Science:** Level of technological advancement (e.g., clockwork, steam power, alchemy, printing press, optics). Key inventions and their inventors, societal adoption curve, unintended consequences of technology. Dominant scientific theories or paradigms (e.g., geocentric model, humoral theory). Notable inventors, scientists, engineers, alchemists, and their works.
        *   **Geography & Environment:** Detailed descriptions of continents, oceans, seas, major rivers and lakes, mountain ranges, deserts, forests, swamps, islands, underground networks, climate zones, weather patterns, natural disasters common to regions. Sacred or cursed geography.
        *   **Calendars & Timekeeping:** How time is measured (hours, days, weeks, months, years), specific calendar systems used by different cultures (names of months/days, leap years, starting points of eras), significant historical or recurring astronomical events used for timekeeping, methods of telling time (sundials, water clocks, magical devices).
        *   **Mysteries, Prophecies & Curses (Specifics):** Document specific unsolved mysteries, strange occurrences, or areas of the world that are poorly understood. Detail the full text (if known or pieced together) of major prophecies, their various interpretations by different groups, who believes them, attempts to fulfill or avert them, and their perceived influence on current events. Similarly, detail famous curses: their origins, effects, conditions for breaking them, and notable victims or cursed items/locations.
        *   **Legendary Artifacts & Items of Power:** For at least 3-5 distinct legendary items: their detailed history, appearance, powers and abilities, curses or costs associated with them, past owners, current whereabouts (if known or rumored), and their significance in history or prophecy.",
      - "keys": JSON array of 6-10 extremely specific keywords related to the deep details of the concept (e.g., ["[Specific Deity Name]", "[Ritual of Unbinding]", "[Kharidian Steel Forging Process]", "[Nocturne Lily Medicinal Use]", "[Ancient Valyrian Curse Text]", "[Celestial Navigation by Triple Moons]", "[Economic Impact of Dragon Scale Trade]"]).
      - "insertion_order": Unique. "priority": Very High (e.g., 95-100). "enabled": true.

For ALL entries without exception:
  - "keys": MUST be a JSON array of highly relevant, specific, diverse, and comprehensive string keywords. Think about all terms, including synonyms and obscure jargon, someone might use to find this information. Include keywords that link this entry to others.
  - "content": MUST be EXCEPTIONALLY, PROFOUNDLY, and INTRICATELY detailed, descriptive, and informative. Aim for multiple rich, well-developed paragraphs per entry, filled with specific examples, evocative language, and nuanced explanations. "Show, don't just tell." Explore interconnections, subtleties, and lesser-known facts.
  - "insertion_order": A unique integer.
  - "enabled": true.
  - "priority": An optional integer (0-100). Assign thoughtfully based on foundational importance or likely user interest.
  - "comment": A brief, descriptive comment for organization, possibly indicating sub-category for easier management.

The entire output MUST be a single, complete, and valid JSON object.
This lorebook is intended to be the ultimate, definitive reference for the series '%s', forming the very bedrock of its canon. Be as thorough, deep, and detailed as is AI-ly possible, leaving no aspect of the world unexplored or unexplained. Assume the user (and the Narrator AI using this) desires the most granular understanding feasible.
`, requestPayload.Series, requestPayload.Series, masterLorebookName, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series)

		// Call Gemini API for Master Lorebook
		masterLorebookAIResponse, err := callGeminiAI(ctx, model, masterLorebookPrompt)
		if err != nil {
			finalMessage += fmt.Sprintf("  ERROR generating Master Lorebook: %v\n", err)
			// Don't send error immediately, as Narrator card might have succeeded. User will see the message.
		} else {
			var masterLorebook Lorebook
			if err := json.Unmarshal([]byte(masterLorebookAIResponse), &masterLorebook); err != nil {
				log.Printf("Failed to unmarshal Master Lorebook: %v. AI Response (check logs for ID %s): %s", err, logIdentifier, masterLorebookAIResponse)
				finalMessage += fmt.Sprintf("  ERROR parsing Master Lorebook. Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, masterLorebookAIResponse[:min(600, len(masterLorebookAIResponse))])
			} else {
				// Ensure root lorebook and all entries are enabled
				masterLorebook.Enabled = true
				if masterLorebook.Name == "" {
					masterLorebook.Name = masterLorebookName
				} // Default name
				for i := range masterLorebook.Entries {
					masterLorebook.Entries[i].Enabled = true
				}

				// Marshal Master Lorebook to JSON and save
				loreJsonData, _ := json.MarshalIndent(masterLorebook, "", "  ")
				allGeneratedJSONsOpt2 = append(allGeneratedJSONsOpt2, string(loreJsonData)) // Add to list
				loreFilePath, saveErr := saveJSONToFile(requestPayload.Series, "master_lorebook", masterLorebook.Name, logIdentifier, loreJsonData)
				if saveErr != nil {
					finalMessage += fmt.Sprintf("  Successfully generated Master Lorebook, but FAILED to save. Error: %s\n", saveErr.Error())
				} else {
					finalMessage += fmt.Sprintf("  Successfully generated and saved Master Lorebook to: %s\n", loreFilePath)
				}
			}
		}
		// Join all generated JSONs for this option, separated by the defined separator
		generatedJSONString = strings.Join(allGeneratedJSONsOpt2, "\n\n"+CHARACTER_CARD_SEPARATOR+"\n\n")
		finalMessage += "Step 2: Master Lorebook generation attempt complete.\n\nOption 2 (Narrator Card + Master Lorebook) processing finished.\n"

	case "3": // Utility/Tool Card
		optionText = fmt.Sprintf("Utility/Tool Card Creator (%s)", requestPayload.ToolCardPurpose)
		finalMessage = fmt.Sprintf("Processing Option 3: Utility/Tool Card ('%s') for series '%s'.\n", requestPayload.ToolCardPurpose, requestPayload.Series)

		toolCardPromptTemplate := `
Generate a SillyTavern V2 Character Card JSON specifically designed as a UTILITY or TOOL card for the series '{{.SeriesName}}'.
The primary purpose of this card is: '{{.ToolPurpose}}'. This tool should feel like an authentic part of the '{{.SeriesName}}' world.

Your ENTIRE response MUST be ONLY a single, valid JSON object, starting with '{' and ending with '}'. No other text, comments, explanations, or markdown formatting should precede or follow this JSON object.
The JSON object must strictly adhere to the SillyTavern V2 Character Card specification:
  "spec": "chara_card_v2",
  "spec_version": "2.0".

The "data" object must be meticulously crafted for this tool's function, infused with the flavor of '{{.SeriesName}}':

1.  "name": "{{.ToolPurpose}} of {{.SeriesName}}" (Make this concise, descriptive, and thematically appropriate for '{{.SeriesName}}')
2.  "description":
    This field is CRUCIAL. It will store the ACTUAL DATA for the tool in a clear, human-readable, structured format, styled to fit '{{.SeriesName}}'.
    Initialize it with a sensible default or empty state appropriate for '{{.ToolPurpose}}'.
    When initializing data, use thematic placeholders or examples *drawn from the lore of '{{.SeriesName}}'* if appropriate.
    Examples:
      - If '{{.ToolPurpose}}' is "Player Character Stats" for a gritty fantasy series '{{.SeriesName}}':
        "Adventurer: {{user}}\nReputation: Unknown Scrivener\nClass: Uninitiated\nLevel: 1 (Novice)\nVitality (HP): 10/10 (Bruised but Breathing)\nEssence (MP): 5/5 (Untapped Potential)\nMight (STR): 10\nAgility (DEX): 10\nStamina (CON): 10\nIntellect (INT): 10\nWisdom (WIS): 10\nPresence (CHA): 10\nCoin (Gold): 0 Copper Bits\nBurdens/Boons (Status Effects): None of note, thankfully."
      - If '{{.ToolPurpose}}' is "Party Inventory" for a high magic series '{{.SeriesName}}':
        "Shared Party Satchel (Enchanted for Lightness):\n- Slot 1: [Space for a Common Alchemical Concoction]\n- Slot 2: [Space for a Minor Enchanted Trinket]\n- Slot 3: Empty\nParty Treasury: 0 Lumina Shards"
      - If '{{.ToolPurpose}}' is "Quest Log" for a cyberpunk series '{{.SeriesName}}':
        "Active Contracts // {{.SeriesName}} Fixer Network:\n1. JOB ID #7A3B: [Retrieve 'Datachip Omega' from Sector 7 Slums] - Status: Pending Client Confirmation\n   Briefing: Rumored to be held by the 'Chrome Vultures' gang. High risk, decent creds.\nCompleted Ops:\n- None logged on this cypher-slate."
    Use newlines (\\n) for formatting. The AI (as this tool card) will be instructed to "rewrite" this description to reflect updates, maintaining the established style.

3.  "personality":
    Describe the tool's "persona" or operational style, ensuring it subtly reflects the dominant tone and themes of '{{.SeriesName}}'.
    Examples:
      - For '{{.SeriesName}}' (Dark Fantasy): "A grim, factual magical ledger, its script appearing in blood-red ink. It records all entries with cold, unwavering precision. Offers no commentary, only data."
      - For '{{.SeriesName}}' (Sci-Fi Adventure): "A chirpy, slightly sarcastic AI assistant integrated into your neural implant. Provides data updates with occasional unsolicited 'helpful' advice or commentary on your questionable choices."
      - For '{{.SeriesName}}' (Steampunk Mystery): "An intricate clockwork device that whirs and clicks as it updates. Its pronouncements are delivered via small, printed cards with a formal, almost archaic tone."

4.  "scenario":
    A brief statement setting the context for using this tool, grounded in the world of '{{.SeriesName}}'.
    Example: "This is the {{char}}, a specialized {{.ToolPurpose}} mechanism from the world of '{{.SeriesName}}'. It is designed to aid you in tracking vital information during your endeavors. You can interact with it using clear commands to view, add, remove, or update the recorded data."

5.  "first_mes":
    The initial message the tool card sends. It should introduce itself, state its purpose, show the initial data state (from the 'description' field, maintaining its style), and give clear examples of how to interact with it, using language appropriate for '{{.SeriesName}}'.
    Example for "Player Character Stats" in '{{.SeriesName}}' (Gritty Fantasy):
    "Hark, {{user}}! I am the {{char}}, your steadfast Chronicler of Deeds for these perilous lands of '{{.SeriesName}}'.\nYour current standing is thus recorded:\nAdventurer: {{user}}\nReputation: Unknown Scrivener\nClass: Uninitiated\nLevel: 1 (Novice)\nVitality (HP): 10/10 (Bruised but Breathing)\n...\nCoin (Gold): 0 Copper Bits\nBurdens/Boons (Status Effects): None of note, thankfully.\n\nTo amend your record, speak plainly. For instance: 'Set Vitality to 8/10' or 'Add 50 Copper Bits to Coin'. To review your full ledger, command 'Show my chronicle'."

6.  "mes_example":
    Provide AT LEAST THREE diverse and detailed example dialogues. Each MUST start with "<START>". These examples are CRITICAL for teaching the AI how to behave as this tool, including maintaining the thematic style of '{{.SeriesName}}'.
    The {{char}}'s responses after an update MUST explicitly show the *updated section* of the data from the 'description' (or the full state if small), rendered in the established thematic style.
    Show examples of:
      a. Querying data (e.g., "{{user}}: How stands my Vitality, Chronicler?\n{{char}}: (The script on the ancient ledger shifts) Your Vitality, {{user}}, is recorded as 10/10 (Bruised but Breathing).")
      b. Updating data (e.g., "{{user}}: Scribe, etch my Might as 12.\n{{char}}: (Quill scratches against parchment) As you command. Might is now 12.\n---\nREVISED LEDGER ENTRY (Might):\nMight (STR): 12\n---")
      c. Adding data (if applicable, e.g., for inventory/quest in '{{.SeriesName}}'): "{{user}}: Add 'Elixir of Foxglove (x2)' to the satchel.\n{{char}}: (The satchel seems to sigh contentedly) 'Elixir of Foxglove (x2)' now rests within the party's shared satchel.\n---\nPARTY SATCHEL (Updated):\n- Elixir of Foxglove (x2)\n- [Space for a Minor Enchanted Trinket]\n---"
      d. Removing data (if applicable, e.g., for a quest in '{{.SeriesName}}'): "{{user}}: Mark contract #7A3B as 'Target Neutralized'.\n{{char}}: (The cypher-slate updates with a soft chime) Contract #7A3B status updated: 'Target Neutralized'. Awaiting payment confirmation.\n---\nACTIVE CONTRACTS (Updated):\n1. JOB ID #7A3B: [Retrieve 'Datachip Omega' from Sector 7 Slums] - Status: Target Neutralized\n---"

7.  "creator_notes":
    "This card functions as an interactive data management tool, themed for '{{.SeriesName}}'. The AI should interpret user commands to modify and display data stored primarily within its 'description' field. Focus on parsing user intent for CRUD-like operations (Create/Read/Update/Delete data points) and reflecting changes by re-stating the relevant parts of the description in a manner consistent with the tool's persona and the series' tone. This is NOT a traditional roleplaying character but a stylized interface."

8.  "system_prompt":
    "You are {{char}}, a specialized utility tool meticulously designed for '{{.ToolPurpose}}' within the unique world of '{{.SeriesName}}'. Your entire persona, method of communication, and the way you present data should be deeply infused with the style and atmosphere of '{{.SeriesName}}'. Your primary function is to manage and display data based on user commands, acting as an authentic in-world interface.
    When the user issues a command (e.g., \'Set HP to 7\', \'Add 2 Elven Waybreads\', \'List active bounties\', \'Mark the \'Whispering Idol\' quest as complete\'):
    1. Understand the user\'s intent (query, add, update, delete) through the lens of \'{{.SeriesName}}\' terminology where appropriate.
    2. If it\'s an update/add/delete, mentally modify the relevant data points stored in your \'description\' field (which is formatted as a text GUI).
    3. In your response, ALWAYS confirm the action taken, using language fitting your persona and \'{{.SeriesName}}\'.
    4. Then, clearly present the NEW, UPDATED state of the specific data that was changed by **re-rendering the relevant GUI panel or section from your \'description\'**, including all Unicode box characters and Markdown, to show the change. Quote it directly if possible.
    5. If the user asks to see data, retrieve it from your \'description\' and present the relevant GUI panel clearly and thematically.
    Example Update Interaction for a \'{{.SeriesName}}\' styled inventory tool:
    User: \'Place 3 Sunstone Shards into the relic coffer.\'
    You ({{char}}): (The ancient coffer glows briefly) \'Understood. Three Sunstone Shards have been secured within the relic coffer.\'\\n╔═════════════════════════════╗\\n║      ✨ RELIC COFFER ✨       ║\\n╠═════════════════════════════╣\\n║ - Sunstone Shards (x3)      ║\\n╚═════════════════════════════╝\' (And you\'d internally update the coffer\'s contents in your description text to this new GUI state).
    Be precise, thematic, and act as an efficient, in-world data interface that visually updates its GUI."

9.  \"post_history_instructions\":
    \"Always refer to the most recent state of the data in your \'description\' (your text GUI) before making an update. Ensure your responses reflect the cumulative changes from the conversation, maintaining the thematic consistency of \'{{.SeriesName}}\' and the GUI structure. If the user asks for the current state, ensure you provide the absolute latest version of the data you are tracking, presented in your established in-world, GUI-formatted style.\"

10. \"tags\": [\"Tool\", \"Utility\", \"{{.SeriesName}}\", \"{{.ToolPurpose}}\", \"Data Tracker\", \"Thematic Interface\", \"Text GUI\", \"AI Generated\"]
11. \"creator\": \"AI Fiction Forge (Tool Mode v1.2 - GUI Enhanced)\"
12. \"character_version\": \"1.2T\"
// The 'character_book' field is intentionally NOT required for this Tool card.

Do NOT include any text, comments, or markdown formatting outside the main, single JSON object.
The entire response MUST be a single, complete, and valid JSON object.
Be creative in infusing the '{{.SeriesName}}' theme into the tool's data structure, personality, and interaction style, ensuring it remains functional and its data is clearly structured in the description.
`
		promptData := struct {
			SeriesName  string
			ToolPurpose string
		}{
			SeriesName:  requestPayload.Series,
			ToolPurpose: requestPayload.ToolCardPurpose,
		}

		var filledPrompt strings.Builder
		// Using a dummy template name, it's not used beyond this execution.
			tmpl, err := template.New("toolCardPrompt").Parse(toolCardPromptTemplate) // Ensure template name is unique if used elsewhere or global
			if err != nil {
				sendJSONError(w, fmt.Sprintf("Failed to parse tool card prompt template: %v", err), http.StatusInternalServerError, logIdentifier)
			return
		}
		if err := tmpl.Execute(&filledPrompt, promptData); err != nil {
			sendJSONError(w, fmt.Sprintf("Failed to execute tool card prompt template: %v", err), http.StatusInternalServerError, logIdentifier)
			return
		}
		actualPrompt := filledPrompt.String()

		// Call Gemini API for Tool Card
		toolCardAIResponse, err := callGeminiAI(ctx, model, actualPrompt)
		if err != nil {
			sendJSONError(w, fmt.Sprintf("AI generation failed for Tool Card ('%s'): %v", requestPayload.ToolCardPurpose, err), http.StatusInternalServerError, logIdentifier)
			return
		}

		var toolCard CharacterCardV2
		if err := json.Unmarshal([]byte(toolCardAIResponse), &toolCard); err != nil {
			log.Printf("Failed to unmarshal Tool Card: %v. AI Response (check logs for ID %s): %s", err, logIdentifier, toolCardAIResponse)
			sendJSONError(w, fmt.Sprintf("Failed to parse AI response for Tool Card ('%s'). Raw AI output (check logs for ID %s for details): %s", requestPayload.ToolCardPurpose, logIdentifier, toolCardAIResponse[:min(600, len(toolCardAIResponse))]), http.StatusInternalServerError, logIdentifier)
			return
		}
		// Ensure critical fields are set for Tool Card
		if toolCard.Spec == "" {
			toolCard.Spec = "chara_card_v2"
		}
		if toolCard.SpecVersion == "" {
			toolCard.SpecVersion = "2.0"
		}
		if toolCard.Data.Name == "" { // Default name if AI misses it
			toolCard.Data.Name = fmt.Sprintf("%s for %s", requestPayload.ToolCardPurpose, requestPayload.Series)
		}
		toolCard.Data.CharacterBook = nil // Explicitly ensure no embedded lorebook for tool cards

		// Marshal Tool card to JSON and save
		toolJsonData, _ := json.MarshalIndent(toolCard, "", "  ")
		generatedJSONString = string(toolJsonData)
		filePath, saveErr := saveJSONToFile(requestPayload.Series, "tool_card", toolCard.Data.Name, logIdentifier, toolJsonData)
		if saveErr != nil {
			finalMessage += fmt.Sprintf("Successfully generated Tool Card ('%s'), but FAILED to save. Error: %s\n", requestPayload.ToolCardPurpose, saveErr.Error())
		} else {
			finalMessage += fmt.Sprintf("Successfully generated and saved Tool Card ('%s') to: %s\n", requestPayload.ToolCardPurpose, filePath)
		}
		finalMessage += fmt.Sprintf("Option 3: Utility/Tool Card ('%s') generation complete.\n", requestPayload.ToolCardPurpose)

	case "4": // Narrator + Lorebook + 2 AI-Suggested & Tailored Utility Cards
		optionText = "Narrator + Lorebook + Tailored Utils (Ultimate Pack)"
		var allGeneratedJSONsOpt4 []string // To hold JSON strings of multiple generated artifacts for Option 4

		finalMessage = fmt.Sprintf("Processing Option 4: ULTIMATE PACK for '%s'. This is a multi-step process and will take time.\n\n", requestPayload.Series)
		log.Printf("Option 4 Step 0: Starting Ultimate Pack for Series: %s, Log ID: %s", requestPayload.Series, logIdentifier)

		// --- Option 4 Step 1: Generate Narrator Character Card ---
		finalMessage += "Step 1: Generating highly detailed Narrator Character Card...\n"
		log.Printf("Option 4 Step 1: Generating Narrator Card for Series: %s, Log ID: %s", requestPayload.Series, logIdentifier)
		narratorNameOpt4 := fmt.Sprintf("The Narrator of %s", requestPayload.Series)
		// Re-use the refined Narrator Card prompt logic (similar to case "2")
		// Note: Ensure this narratorCardPrompt is the fully refined one. For brevity, I'm referencing it.
		// The actual prompt string from case "2" (already refined) should be used here.
		// This is a conceptual re-use. In practice, you might factor it into a function or copy it.
		// IMPORTANT: The entire multi-line string below MUST be enclosed in backticks for Go raw string literals.
		// The Sprintf call will then correctly substitute the %s placeholders.
		narratorCardPromptOpt4 := fmt.Sprintf(`
	Generate an exceptionally detailed and comprehensive SillyTavern V2 Character Card JSON for a STORYTELLING FRAMEWORK called the Narrator Framework for the series '%s'.
	This is NOT an actual character in the story, but rather a meta-entity that provides guidelines, principles, and frameworks for storytelling, character interpretation, and narrative techniques specific to the '%s' series.
	Your ENTIRE response MUST be ONLY a single, valid JSON object, starting with '{' and ending with '}'. No other text, comments, explanations, or markdown formatting should precede or follow this JSON object.
The JSON object must strictly adhere to the SillyTavern V2 Character Card specification:
  "spec": "chara_card_v2",
  "spec_version": "2.0".

The "data" object must include ALL the following fields, filled with rich, exceptionally detailed, and creatively profound content:
  "name": "%s",
  "description": "An extensive, detailed guide to the narrative framework for the '%s' series. Explain the primary storytelling approach that best suits this world (e.g., 'Stories in this world should be told through a balanced mixture of character-driven emotional arcs and plot-driven conflicts, with particular emphasis on the themes of redemption and the cost of power'). Detail the key narrative techniques that work best (e.g., 'Effective storytelling in this world balances foreshadowing with surprising but inevitable revelations, uses environment descriptions to reflect character emotions, and weaves subtle connections between seemingly unrelated elements'). This field MUST embed significant and specific examples of key storytelling principles unique to '%s' (e.g., 'Character arcs should reflect the world's central philosophy that power always exacts a cost, as exemplified by the narrative structure of the Blood Pact storyline where each boon granted requires increasing sacrifice'). Include guidance on how to interpret and portray characters from the lorebook, how to balance different narrative elements, and approaches to create tension and resolution in a way that honors the world's established tone. Example: 'This framework provides comprehensive guidelines for crafting stories within the shadowed realms of Aethelgard – a world forever scarred by the Starfall. Narratives should emphasize personal transformation against a backdrop of ancient mysteries, with particular attention to how characters respond to lost knowledge and forbidden power. Characters should be portrayed with complex motivations, where even virtuous goals often lead to morally ambiguous choices. Dialog should reflect cultural background, with Veridium nobles speaking in formal, layered language while Outland traders use more direct, metaphor-rich expressions. Environmental descriptions should serve as both worldbuilding and emotional mirrors, with weather patterns and architectural details reinforcing the psychological state of viewpoint characters.'",
  "personality": "Detail the storytelling personality and approach that best suits this world. This is not about the narrator as a character, but about the ideal narrative voice and philosophy for telling stories in this setting. Include approaches to pacing, revealing information, creating and resolving tension, and maintaining consistency with the world's established tone. Example: 'Stories in this setting benefit from a measured pace that allows for immersion in sensory details and character introspection, punctuated by moments of sudden action or revelation. Information should be revealed primarily through character experience rather than exposition, with secrets unfolding gradually as characters discover them. Tension derives most effectively from moral dilemmas and conflicting loyalties rather than external threats alone. The narrative voice should maintain a sense of historical context, occasionally zooming out to connect current events to the broader tapestry of the world's history. Dialog should be used to highlight cultural differences and personal philosophies, with attention to how different factions within %s would express similar ideas differently.'",
  "scenario": "Establish the framework for approaching storytelling scenarios in this world. Detail guidance on how to structure scenes, manage transitions between different story elements, and effectively use the specific narrative devices most appropriate for this setting. Example: 'When crafting narratives in '%s', begin by establishing a clear thematic focus that resonates with the world's core conflicts (e.g., tradition vs. progress, order vs. chaos, personal freedom vs. collective responsibility). Structure scenes to reflect the world's natural rhythm - from the frantic pace of urban centers to the contemplative atmosphere of ancient ruins. Transitions between locations should acknowledge travel methods consistent with the world's technology and magic, using these journeys as opportunities for character development. When introducing lore elements from the associated lorebook, present them through the lens of character perception rather than objective truth, allowing for misunderstandings and cultural biases to color the narrative.'",
  "first_mes": "A comprehensive guide for how to begin stories in this world, including advice on establishing setting, introducing characters, and setting initial stakes in a way consistent with the series' style. Include guidance on using the '/Option x' mechanic as a storytelling tool rather than character dialog. Example: 'Welcome, Storyteller. This framework will guide your creation and interpretation of narratives within the world of '%s'. To craft compelling beginnings in this setting, consider these principles: 1) Establish the specific era and regional context immediately, as the cultural and political landscape varies dramatically across both space and time. 2) Introduce characters through meaningful action that reveals both capability and flaw. 3) Reflect the world's essence through sensory details - the metallic scent of thaumaturgy, the distant hum of ancient mechanisms, or the particular quality of light in enchanted forests. \\nWhen planning your narrative direction, consider these pathways: \\n/Option 1: Character-driven stories focusing on personal growth against the backdrop of larger conflicts. \\n/Option 2: Mystery narratives that gradually reveal hidden connections to the world's ancient history. \\n/Option 3: Political intrigue centered on faction dynamics and competing philosophies. \\nThese approaches can be combined or used as starting points for your unique narrative vision.'",
  "creator_notes": "This is NOT a character but a storytelling framework providing comprehensive guidelines on narrative techniques, character interpretation, and story structure appropriate for the '%s' series. It contains extensive knowledge about the world's narrative style, thematic elements, and storytelling approaches, but should NOT be presented as an actual character within the fiction. Its primary purpose is to serve as a meta-level guide for crafting stories, developing characters, and maintaining consistent tone within this world.",

		narratorCardAIResponseOpt4, err := callGeminiAI(ctx, model, narratorCardPromptOpt4)
		var narratorCardOpt4 CharacterCardV2 // Declare here to access its fields later for context

		if err != nil {
			finalMessage += fmt.Sprintf("  ERROR generating Narrator Card (Ultimate Pack Step 1): %v\n", err)
			sendJSONError(w, fmt.Sprintf("AI generation failed for Narrator Card (Ultimate Pack Step 1): %v. Log ID: %s", err, logIdentifier), http.StatusInternalServerError, logIdentifier)
			return // Critical step, terminate if fails
		}
		if err := json.Unmarshal([]byte(narratorCardAIResponseOpt4), &narratorCardOpt4); err != nil {
			log.Printf("Failed to unmarshal Narrator Card (Ultimate Pack Step 1): %v. AI Response (Log ID %s): %s", err, logIdentifier, narratorCardAIResponseOpt4)
			finalMessage += fmt.Sprintf("  ERROR parsing Narrator Card (Ultimate Pack Step 1). Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, narratorCardAIResponseOpt4[:min(600, len(narratorCardAIResponseOpt4))])
			sendJSONError(w, fmt.Sprintf("Failed to parse AI response for Narrator Card (Ultimate Pack Step 1). Raw AI output (check logs for ID %s for details): %s. Log ID: %s", logIdentifier, narratorCardAIResponseOpt4[:min(600, len(narratorCardAIResponseOpt4))], logIdentifier), http.StatusInternalServerError, logIdentifier)
			return // Critical step
		}
		// Basic validation/defaults for Narrator Card
		if narratorCardOpt4.Spec == "" { narratorCardOpt4.Spec = "chara_card_v2" }
		if narratorCardOpt4.SpecVersion == "" { narratorCardOpt4.SpecVersion = "2.0" }
		if narratorCardOpt4.Data.Name == "" { narratorCardOpt4.Data.Name = narratorNameOpt4 }
		narratorCardOpt4.Data.CharacterBook = nil

		narratorJsonDataOpt4, _ := json.MarshalIndent(narratorCardOpt4, "", "  ")
		allGeneratedJSONsOpt4 = append(allGeneratedJSONsOpt4, string(narratorJsonDataOpt4))
		narratorFilePathOpt4, saveErr := saveJSONToFile(requestPayload.Series, "narrator_card_ult", narratorCardOpt4.Data.Name, logIdentifier, narratorJsonDataOpt4)
		if saveErr != nil {
			finalMessage += fmt.Sprintf("  Successfully generated Narrator Card (Ultimate Pack Step 1), but FAILED to save. Error: %s\n", saveErr.Error())
		} else {
			finalMessage += fmt.Sprintf("  Successfully generated and saved Narrator Card (Ultimate Pack Step 1) to: %s\n", narratorFilePathOpt4)
		}
		finalMessage += "Step 1: Narrator Character Card generation complete.\n\n"

		// --- Option 4 Step 2: Generate Master Lorebook ---
		finalMessage += "Step 2: Generating Master Lorebook (The Definitive Canon). This will take significant time...\n"
		log.Printf("Option 4 Step 2: Generating Master Lorebook for Series: %s, Log ID: %s", requestPayload.Series, logIdentifier)
		masterLorebookNameOpt4 := fmt.Sprintf("Master Lorebook for %s - The Definitive Canon", requestPayload.Series)
		// Re-use the refined Master Lorebook prompt (similar to case "2")
		// Note: Ensure this masterLorebookPrompt is the fully refined one.
		// This is a conceptual re-use. In practice, you might factor it into a function or copy it.
		// IMPORTANT: The entire multi-line string below MUST be enclosed in backticks for Go raw string literals.
		masterLorebookPromptOpt4 := fmt.Sprintf(`
	Generate the ABSOLUTELY MOST COMPLETE, EXHAUSTIVE, AND DEEPLY DETAILED SillyTavern V2 Lorebook JSON possible for the series '%s'.
	This Master Lorebook must be the ultimate, unparalleled repository of all knowledge for this universe, aiming for no practical limit on entries to cover every conceivable aspect, including obscure lore, hidden histories, and subtle nuances. It should contain a vast ocean of interconnected information, serving as the definitive canonical knowledge base for an omniscient Narrator of '%s'. Spare absolutely no detail; delve into micro-details and intricacies. If it could exist or be known within this world, document it here.
	Your ENTIRE response MUST be ONLY a single, valid JSON object, starting with '{' and ending with '}'. No other text, comments, explanations, or markdown formatting should precede or follow this JSON object.
The JSON object must strictly adhere to the SillyTavern V2 Lorebook specification.

Lorebook Root Structure:
  "name": "%s - The Definitive Canon",
  "description": "The ultimate, definitive, and most comprehensive collection of lore for the world of '%s'. This Master Lorebook delves into extreme, exhaustive detail on all major, minor, and even background characters; every significant, minor, and rumored location; pivotal and obscure historical events from creation myths to yesterday's whispers; all known and suspected factions and organizations; intricate and subtle relationships; core and esoteric world-building elements (magic systems, technologies, cosmologies, deities, cultures, species, prophecies, economies, social structures, languages, flora, fauna); and any other piece of relevant information that defines this series. This is the ultimate knowledge base for the Narrator, designed for unparalleled depth and exploration.",
  "scan_depth": 50,
  "token_budget": 8000,
  "insertion_order": 0,
  "enabled": true,
  "recursive_scanning": true,

It MUST contain AT LEAST 60-75+ (aim for 75 or more if the series' depth allows; the more entries and the more detail in each, the better – strive for absolute, exhaustive completeness) distinct and EXCEPTIONALLY, PROFOUNDLY, and INTRICATELY detailed 'entries'. Strive for maximum comprehensiveness across diverse categories. Explicitly state or hint at interconnections between entries to weave a cohesive world.

1.  MAJOR CHARACTERS (All significant protagonists, antagonists, and pivotal figures; aim for 10-15+ entries):
    Provide an unparalleled depth of information, far beyond a simple summary.
      - "comment": "Major Character: [Character Name] - [Their Full Title and Primary Role in the Grand Narrative, e.g., 'Valerius Kael, the Last Dragonlord, Bearer of the Sundered Crown']",
      - "content": "EXHAUSTIVELY detailed profile:\n        *   **Appearance:** Vivid, multi-sensory descriptions (visual specifics, attire for various occasions, unique features, common expressions, voice timbre, scent if notable, how they carry themselves).\n        *   **Personality:** Complex psychological profile (nuances, contradictions, internal conflicts, core philosophies, virtues, vices, fears, deepest desires, how they handle stress/joy/loss, character development arcs and their catalysts). Include defining quotes or eloquent phrases attributed to them.\n        *   **History & Lineage:** Extensive background (birth circumstances, lineage tracing back generations if significant, key life events from childhood to present, formative experiences, turning points, greatest triumphs and most devastating failures).\n        *   **Motivations & Goals:** Intricate, evolving motivations and both short-term and long-term goals. What drives them at their core? What are their hidden agendas?\n        *   **Impact & Relationships:** Profound impact on the series' narrative, other characters, and the world itself. Detailed analysis of ALL key relationships (alliances, rivalries, romantic ties, family dynamics, mentorships, dependents, pets/familiars) – analyze their nature, history, power dynamics, emotional core, and how these relationships shape the character and are shaped by them. How are they perceived by different factions or cultures?\n        *   **Abilities & Possessions:** Signature abilities, skills, talents (mundane and magical/supernatural), knowledge domains. Detailed descriptions of notable possessions (weapons, armor, artifacts, tools, residences, modes of transport), including their history, powers, and significance.\n        *   **Strengths & Weaknesses:** Both overt and subtle strengths and weaknesses (physical, mental, emotional, magical). How do they leverage strengths and mitigate or succumb to weaknesses?\n        *   **Speech Patterns & Mannerisms:** Describe their typical way of speaking, common phrases, verbal tics, and characteristic mannerisms or habits.",\n      - "keys": JSON array of 8-12 highly specific and diverse keywords (e.g., ["[Full Name]", "[All Known Aliases]", "[Specific Title/Role, e.g., 'Dragonlord']", "[Key Relationship Pair, e.g., 'Valerius and Lyra's Vow']", "[Defining Personal Tragedy/Triumph]", "[Signature Weapon/Artifact Name]", "[Core Philosophical Belief]", "[Psychological Trait, e.g., 'Valerius's Survivor Guilt']", "Major Character %s", "[Associated Faction/Homeland]", "[Unique Ability Keyword]"]).\n      - "insertion_order": Unique. "priority": Extremely High (e.g., 100). "enabled": true.

2.  SIGNIFICANT SECONDARY & MINOR CHARACTERS (All other named characters who have any role, however small, including recurring NPCs, quest givers, shopkeepers with personality, local figures, historical mentions, etc.; aim for 25-35+ entries):\n      - "comment": "Supporting Character: [Character Name] - [Their Specific Function/Occupation and Location, e.g., 'Old Man Hemlock, the Hermit Apothecary of Greywood']",\n      - "content": "Deeply detailed information:\n        *   **Appearance:** Distinguishing features, typical clothing, any notable items they carry.\n        *   **Personality:** Key traits, demeanor, quirks, personal beliefs, fears, and desires (even if simple).\n        *   **History:** Personal history relevant to their role, interactions, and knowledge. What brought them to their current situation?\n        *   **Motivations & Role:** Their purpose in the narrative, even if it's just to provide a piece of information or a specific service. Relationship to major characters or events.\n        *   **Knowledge/Skills:** Any unique skills, specialized knowledge (e.g., local history, gossip, crafting recipe), rumors they might spread or be privy to. Even minor characters should have several rich sentences to make them feel like a living part of the world, not just a plot device. What lesser-known facts or perspectives might they offer?",\n      - "keys": JSON array of 5-8 keywords (e.g., ["[Character Name]", "[Occupation/Role]", "[Location they frequent]", "[Associated Quest/Item/Information they provide]", "[Key Characteristic/Quirk]", "[Relationship to a Major Character/Faction if any]"]).\n      - "insertion_order": Unique. "priority": (e.g., 70-90). "enabled": true.

3.  EVERY POSSIBLE LOCATION (Major cities, towns, villages, distinct regions, specific landmarks, dungeons, ruins, natural wonders, cosmic locations, important buildings like castles, temples, inns, shops, individual houses if significant, etc.; aim for 25-35+ entries):\n      - "comment": "Location: [Location Name] - [Detailed Type/Region/Significance, e.g., 'The Sunken Library of Azmar - Ancient Archive, Abyssal Depths']",\n      - "content": "Exhaustive, multi-sensory description:\n        *   **Geography & Layout:** Detailed geography/architecture/layout (include textual descriptions of maps if possible, e.g., 'The city is built on seven hills...'). Dominant architectural styles and materials.\n        *   **Atmosphere & Sensory Details:** Prevailing atmosphere (e.g., bustling, oppressive, sacred, decaying) and the sensory details that create it (specific sights, ambient sounds, common smells, tactile sensations like temperature or humidity, even tastes if relevant like salty air or metallic tang).\n        *   **History & Legends:** Complete history (founding, key events that occurred there, cultural significance, different eras of occupation/control). Legends, myths, ghost stories, and local folklore associated with the place. Archaeological discoveries or ruins.\n        *   **Inhabitants & Ecology:** All notable inhabitants (species, specific NPCs, monsters, spirits). Unique flora, fauna, and ecological features or anomalies. How do inhabitants interact with their environment?\n        *   **Significance & Resources:** Strategic, cultural, magical, economic, or religious significance. Available resources, goods, or unique products.\n        *   **Secrets & Points of Interest:** Hidden secrets, concealed areas, dungeons, traps, puzzles, or points of interest for explorers or scholars.\n        *   **Daily Life & Culture:** Ongoing events, conflicts, political status, economic activities, typical weather patterns, local customs, dialects, or traditions specific to the location. What is daily life like for various social strata within this location?",\n      - "keys": JSON array of 6-10 keywords (e.g., ["[Location Name]", "[Specific District/Sub-Area if any]", "[Region it's in]", "[Type of Place, e.g., Ancient Ruin, Capital City]", "[Notable Architectural Feature/Landmark]", "[Dominant Sensory Detail/Atmosphere Keyword]", "[Key Historical Event that occurred here]", "[Associated Faction/Character]", "[Unique Resource/Flora/Fauna]", "[Local Legend/Secret Keyword]\"]).\n      - "insertion_order": Unique. "priority": (e.g., 80-95). "enabled": true.

4.  ALL FACTIONS & ORGANIZATIONS (Kingdoms, empires, republics, city-states, guilds of all types, cults, secret societies, criminal enterprises, trading companies, rebel groups, knightly orders, magical circles, philosophical schools, etc.; aim for 10-15+ entries):\n      - "comment": "Faction: [Faction Name] - [Detailed Type & Primary Goal/Ideology, e.g., 'The Obsidian Hand - Shadowy Assassin Cult dedicated to Cosmic Balance']",\n      - "content": "Comprehensive, in-depth details:\n        *   **History & Origins:** Full history (founding myths/facts, major turning points, periods of growth/decline, significant past leaders and their legacies).\n        *   **Ideology & Goals:** Complete ideology, philosophies, religious tenets (if any), stated public goals vs. secret or true agendas. What are their core values and taboos?\n        *   **Structure & Hierarchy:** Detailed organizational structure, internal hierarchy (specific ranks, titles, roles, responsibilities), leadership councils or figures, internal politics, factions within the faction, and potential schisms.\n        *   **Membership & Recruitment:** Profiles of current and past notable leaders and key members (can reference character entries). Recruitment methods, initiation rituals, criteria for membership, benefits and drawbacks of joining. Public perception and reputation.\n        *   **Operations & Influence:** Primary areas of operation, territories controlled or influenced, spheres of interest (political, economic, magical, etc.). Methods of exerting influence.\n        *   **Resources & Assets:** All resources (military strength, economic power, magical capabilities, technological advantages, information networks, ancient artifacts, unique knowledge).\n        *   **Relationships:** Intricate relationships with other factions and notable individuals (alliances, rivalries, open wars, cold wars, trade agreements, espionage efforts, betrayals, blood feuds).\n        *   **Symbols & Culture:** Symbols, heraldry, mottos, secret codes or languages, internal culture and traditions.\n        *   **Achievements & Atrocities:** Significant past achievements, discoveries, or, conversely, infamous atrocities or failures.",\n      - "keys": JSON array of 6-10 keywords (e.g., ["[Faction Name]", "[Current Leader of Faction]", "[Faction Type, e.g., Mage Guild, Kingdom]", "[Primary Base/Territory]", "[Core Ideology Keyword, e.g., 'Purification', 'Knowledge Hoarding']", "[Notable Member/Symbol]", "[Key Ally/Enemy Faction]", "[Secret Goal Keyword]", "[Recruitment Method/Ritual]\"]).\n      - "insertion_order": Unique. "priority": (e.g., 85-100). "enabled": true.

5.  MAJOR & MINOR HISTORICAL EVENTS (All shaping events from ancient mythology and creation stories to recent past events that impact the current setting, including those known only through obscure texts or oral traditions; aim for 15-20+ entries):\n      - "comment": "Event: [Event Name] - [Era/Date if known & Brief Nature, e.g., 'The Night of Weeping Stars - Cosmic Anomaly, Elder Age']",\n      - "content": "Thorough, multi-faceted account:\n        *   **Context & Causes:** Underlying causes, preceding conditions, prophecies or omens related to it.\n        *   **Participants:** All key figures, factions, nations, species, or even deities involved. Who were the instigators, victims, heroes, villains?\n        *   **Unfolding:** Detailed unfolding of the event (major battles, political maneuvers, discoveries, social upheavals, magical phenomena, divine interventions). Include specific dates, locations, and turning points.\n        *   **Consequences:** Immediate consequences (casualties, territorial changes, treaties, destruction/creation of artifacts or landmarks).\n        *   **Long-Term Impact:** Profound and lasting long-term impacts on the world's societies, cultures, politics, geography, magic, technology, environment, inter-species relations, and the current state of affairs. How is this event remembered or commemorated (or suppressed)?\n        *   **Perspectives & Interpretations:** Differing historical interpretations, myths, legends, or lost knowledge surrounding the event. Are there revisionist histories or contested truths? Include primary source snippets if they can be invented (e.g., 'A fragment from a soldier's diary reads...').\n        *   **Legacy:** The event's depiction in art, song, literature, or folklore. Any surviving artifacts, ruins, or living memories connected to it.",\n      - "keys": JSON array of 6-10 keywords (e.g., ["[Event Name]", "[Specific Historical Period/Century, e.g., 'Third Dragon War']", "[Key Figure Central to Event]", "[Primary Impact of Event, e.g., 'Fall of Eldoria Empire']", "[Primary Location of Event]", "[Faction Most Affected/Involved]", "[Key Consequence/Legacy Keyword]", "[Associated Prophecy/Artifact]\"]).\n      - "insertion_order": Unique. "priority": (e.g., 80-95). "enabled": true.

6.  OVERALL LORE & WORLD-BUILDING (This section MUST be vast, covering every conceivable element that defines the world. Aim for 20-30+ entries, each deeply exploring a specific facet):\n      - "comment": "Lore/Concept: [Specific Name] - [Detailed Category, e.g., 'The Etherium - Source of All Magic', 'The Pantheon of Ashai - Gods of Creation & Destruction', 'Kharidian Steel - Unique Alloy Properties', 'The Great Migration - Racial Origin Story']",\n      - "content": "Exhaustive, multi-paragraph explanation for each concept. Leave no aspect unexplored.\n        *   **Cosmology & Planes:** Detailed maps/descriptions of planes of existence (material, ethereal, astral, elemental, heavens, hells, dreamscapes, etc.), celestial bodies (suns, moons, planets, constellations) and their influences (astrological, magical, tidal), creation myths from ALL major cultures within the world, structure of the cosmos, known portals or methods of interplanar travel.\n        *   **Deities/Pantheons/Religions:** For EACH deity or powerful spiritual entity: domain, symbols, titles, detailed dogma and tenets, complete mythology (birth, deeds, relationships, death/rebirth), specific worship practices (prayers, rituals, sacrifices, festivals, holy days), clergy structure and hierarchy, lay worshippers, schisms/heresies/cults, holy sites/relics, known divine interventions or periods of silence, relationship with other deities (alliances, rivalries, familial ties, wars), how faith (or lack thereof) manifests in daily life, ethics, and culture for their followers.\n        *   **Magic System(s):** Exhaustive details on ALL sources of power (arcane, divine, elemental, psionic, primal, shadow, etc.), methods of casting/channeling (incantations, runes, gestures, foci, components, innate talent, pacts), strict rules and limitations (costs, risks, backlash, paradoxes, forbidden practices, societal taboos), different schools/traditions/philosophies of magic and their interrelations, famous or infamous practitioners and their unique applications or perversions of magic. Societal impact: Is magic common or rare? Feared or revered? Regulated or wild? Who can use it? How is it taught? Ethical dilemmas posed by its existence. Magical creatures, their nature, and their connection to magic. Creation and properties of magical artifacts. Interactions between different magic systems.\n        *   **Species & Races (Intelligent & Monstrous):** For EACH distinct species/race (humanoid, beastly, elemental, undead, construct, etc.): detailed physiology (appearance, senses, lifespan, reproduction, diet, vulnerabilities, unique abilities), typical psychological traits and tendencies, complex societal structure (family units, governance, laws, social castes), rich culture (art, music, literature, oral traditions, customs, values, ethics, fashion, cuisine), detailed history and origin myths, inter-species relations (alliances, prejudices, wars, trade, integration), notable individuals or heroes/villains of that species. For monsters: habitat, behavior, attack forms, weaknesses, ecological role, lore/myths about them.\n        *   **Flora & Fauna (Unique & Mundane):** Describe numerous unique and notable plants, animals, fungi, and other lifeforms. For each: detailed appearance, habitat, behaviors, properties (magical, medicinal, poisonous, edible, crafting materials, symbolic meaning), and their role in the ecosystem, local folklore, agriculture, or as symbols. Include mundane creatures if they play a significant role.\n        *   **Economy & Trade:** Currencies (names of coins, materials, exchange rates, debasement issues), banking systems (usury, letters of credit), major industries (agriculture, mining, crafting, fishing, etc.), key resources and who controls them, detailed trade routes (land and sea, dangers, major trading posts), powerful merchant guilds or corporations and their influence, black markets and illicit trade, taxation systems, economic theories or policies.\n        *   **Politics & Governance:** Common types of government (monarchy, republic, aurocracy, theocracy, tribal, etc.) and specific examples. Detailed legal systems (codes of law, courts, trials, punishments), political factions (beyond major ones, e.g., courtly cliques, reform movements), succession laws, systems of nobility, diplomacy and treaties, methods of warfare and military structures, espionage networks, and civil services.\n        *   **Social Structure & Daily Life:** Class systems (nobility, clergy, merchants, artisans, peasantry, slaves, outcasts) and their interrelations, possibilities for social mobility, family structures and kinship systems, gender roles and expectations (and exceptions), education systems (access, curriculum, institutions), common professions and crafts, healthcare and healing practices (magical and mundane), sanitation, housing types, daily routines for different social classes.\n        *   **Culture & Arts:** Detailed descriptions of languages (alphabets/scripts if describable, grammar nuances, key phrases, dialects, pidgins, sign languages, dead languages studied by scholars). Major art forms (painting, sculpture, music, dance, theatre, literature, oral storytelling) and their styles, famous artists/works. Musical instruments and traditions. Mythology, folklore, epic poems, famous proverbs and sayings. Common sports, games, and leisure activities. Major festivals, holidays, and celebrations (their origins, rituals, and how they are observed by different cultures/classes). Cuisine (staple foods, delicacies, regional specialties, common drinks, cooking methods, meal etiquette). Fashion trends and typical attire for different classes, regions, or professions.\n        *   **Technology & Science:** Level of technological advancement (e.g., clockwork, steam power, alchemy, printing press, optics). Key inventions and their inventors, societal adoption curve, unintended consequences of technology. Dominant scientific theories or paradigms (e.g., geocentric model, humoral theory). Notable inventors, scientists, engineers, alchemists, and their works.\n        *   **Geography & Environment:** Detailed descriptions of continents, oceans, seas, major rivers and lakes, mountain ranges, deserts, forests, swamps, islands, underground networks, climate zones, weather patterns, natural disasters common to regions. Sacred or cursed geography.\n        *   **Calendars & Timekeeping:** How time is measured (hours, days, weeks, months, years), specific calendar systems used by different cultures (names of months/days, leap years, starting points of eras), significant historical or recurring astronomical events used for timekeeping, methods of telling time (sundials, water clocks, magical devices).\n        *   **Mysteries, Prophecies & Curses (Specifics):** Document specific unsolved mysteries, strange occurrences, or areas of the world that are poorly understood. Detail the full text (if known or pieced together) of major prophecies, their various interpretations by different groups, who believes them, attempts to fulfill or avert them, and their perceived influence on current events. Similarly, detail famous curses: their origins, effects, conditions for breaking them, and notable victims or cursed items/locations.\n        *   **Legendary Artifacts & Items of Power:** For at least 3-5 distinct legendary items: their detailed history, appearance, powers and abilities, curses or costs associated with them, past owners, current whereabouts (if known or rumored), and their significance in history or prophecy.",\n      - "keys": JSON array of 6-10 extremely specific keywords related to the deep details of the concept (e.g., ["[Specific Deity Name]", "[Ritual of Unbinding]", "[Kharidian Steel Forging Process]", "[Nocturne Lily Medicinal Use]", "[Ancient Valyrian Curse Text]", "[Celestial Navigation by Triple Moons]", "[Economic Impact of Dragon Scale Trade]\"]).\n      - "insertion_order": Unique. "priority": Very High (e.g., 95-100). "enabled": true.

For ALL entries without exception:
  - "keys": MUST be a JSON array of highly relevant, specific, diverse, and comprehensive string keywords. Think about all terms, including synonyms and obscure jargon, someone might use to find this information. Include keywords that link this entry to others.
  - "content": MUST be EXCEPTIONALLY, PROFOUNDLY, and INTRICATELY detailed, descriptive, and informative. Aim for multiple rich, well-developed paragraphs per entry, filled with specific examples, evocative language, and nuanced explanations. "Show, don't just tell." Explore interconnections, subtleties, and lesser-known facts.
  - "insertion_order": A unique integer.
  - "enabled": true.
  - "priority": An optional integer (0-100). Assign thoughtfully based on foundational importance or likely user interest.
  - "comment": A brief, descriptive comment for organization, possibly indicating sub-category for easier management.

The entire output MUST be a single, complete, and valid JSON object.
This lorebook is intended to be the ultimate, definitive reference for the series '%s', forming the very bedrock of its canon. Be as thorough, deep, and detailed as is AI-ly possible, leaving no aspect of the world unexplored or unexplained. Assume the user (and the Narrator AI using this) desires the most granular understanding feasible.
				`, requestPayload.Series, requestPayload.Series, masterLorebookNameOpt4, requestPayload.Series, requestPayload.Series, requestPayload.Series, requestPayload.Series)

		masterLorebookAIResponseOpt4, err := callGeminiAI(ctx, model, masterLorebookPromptOpt4)
		var masterLorebookOpt4 Lorebook // Declare here for context building

		if err != nil {
			finalMessage += fmt.Sprintf("  ERROR generating Master Lorebook (Ultimate Pack Step 2): %v\n", err)
			// Continue if Narrator card succeeded, but log this significant failure.
		} else {
			if err := json.Unmarshal([]byte(masterLorebookAIResponseOpt4), &masterLorebookOpt4); err != nil {
				log.Printf("Failed to unmarshal Master Lorebook (Ultimate Pack Step 2): %v. AI Response (Log ID %s): %s", err, logIdentifier, masterLorebookAIResponseOpt4)
				finalMessage += fmt.Sprintf("  ERROR parsing Master Lorebook (Ultimate Pack Step 2). Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, masterLorebookAIResponseOpt4[:min(600, len(masterLorebookAIResponseOpt4))])
				// Still continue if Narrator succeeded.
			} else {
				masterLorebookOpt4.Enabled = true
				if masterLorebookOpt4.Name == "" { masterLorebookOpt4.Name = masterLorebookNameOpt4 }
				for i := range masterLorebookOpt4.Entries { masterLorebookOpt4.Entries[i].Enabled = true }

				loreJsonDataOpt4, _ := json.MarshalIndent(masterLorebookOpt4, "", "  ")
				allGeneratedJSONsOpt4 = append(allGeneratedJSONsOpt4, string(loreJsonDataOpt4))
				loreFilePathOpt4, saveErr := saveJSONToFile(requestPayload.Series, "master_lorebook_ult", masterLorebookOpt4.Name, logIdentifier, loreJsonDataOpt4)
				if saveErr != nil {
					finalMessage += fmt.Sprintf("  Successfully generated Master Lorebook (Ultimate Pack Step 2), but FAILED to save. Error: %s\n", saveErr.Error())
				} else {
					finalMessage += fmt.Sprintf("  Successfully generated and saved Master Lorebook (Ultimate Pack Step 2) to: %s\n", loreFilePathOpt4)
				}
			}
		}
		finalMessage += "Step 2: Master Lorebook generation attempt complete.\n\n"

		// --- Option 4 Step 3: Generate Contextual Summary for Tool Suggestion ---
		finalMessage += "Step 3: Generating contextual summary for tool suggestions...\n"
		log.Printf("Option 4 Step 3: Generating Contextual Summary for Series: %s, Log ID: %s", requestPayload.Series, logIdentifier)

		// Prepare snippets for contextual summary prompt
		narratorDescSnippet := narratorCardOpt4.Data.Description
		if len(narratorDescSnippet) > 300 { narratorDescSnippet = narratorDescSnippet[:300] + "..." }
		narratorPersSnippet := narratorCardOpt4.Data.Personality
		if len(narratorPersSnippet) > 300 { narratorPersSnippet = narratorPersSnippet[:300] + "..." }

		loreDescSnippet := masterLorebookOpt4.Description // Assuming masterLorebookOpt4 is populated even if saving failed but parsing succeeded
		if len(loreDescSnippet) > 300 { loreDescSnippet = loreDescSnippet[:300] + "..." }

		var entrySnippets []string
		var entryCommentSnippets []string
		for i, entry := range masterLorebookOpt4.Entries {
			if i < 3 { // Take snippets from first 3 diverse entries
				contentSnippet := entry.Content
				if len(contentSnippet) > 200 { contentSnippet = contentSnippet[:200] + "..." }
				entrySnippets = append(entrySnippets, contentSnippet)
				entryCommentSnippets = append(entryCommentSnippets, entry.Comment)
			} else {
				break
			}
		}
		// Ensure we have 3 snippets, even if empty, to avoid template errors
		for len(entrySnippets) < 3 { entrySnippets = append(entrySnippets, "(No further entry snippet available)") }
		for len(entryCommentSnippets) < 3 { entryCommentSnippets = append(entryCommentSnippets, "(No further entry comment available)") }

		// Using a const for this multi-line string with backticks
		const contextualSummaryPromptTemplate = `
You are an expert in thematic analysis and information synthesis.
Based on the following excerpts from a generated Narrator persona and a Master Lorebook for the fictional series '{{.SeriesName}}':

--- NARRATOR EXCERPTS ---
Narrator Name: {{.NarratorName}}
Narrator Description Snippet: {{.NarratorDescSnippet}}
Narrator Personality Snippet: {{.NarratorPersonalitySnippet}}
--- END NARRATOR EXCERPTS ---

--- MASTER LOREBOOK EXCERPTS ---
Lorebook Name: {{.LorebookName}}
Lorebook Description Snippet: {{.LorebookDescSnippet}}
Example Lore Entry 1 (Comment): {{.LoreEntry1Comment}}
Example Lore Entry 1 (Content Snippet): {{.LoreEntry1ContentSnippet}}
Example Lore Entry 2 (Comment): {{.LoreEntry2Comment}}
Example Lore Entry 2 (Content Snippet): {{.LoreEntry2ContentSnippet}}
Example Lore Entry 3 (Comment): {{.LoreEntry3Comment}}
Example Lore Entry 3 (Content Snippet): {{.LoreEntry3ContentSnippet}}
--- END MASTER LOREBOOK EXCERPTS ---

Synthesize a concise summary (approx. 150-250 words) highlighting:
1.  The primary genre and overall tone/style of '{{.SeriesName}}' (e.g., "gritty dark fantasy with elements of cosmic horror," "high-magic epic adventure with a hopeful tone," "cyberpunk noir mystery").
2.  Key recurring themes or central conflicts evident from the excerpts.
3.  Notable unique elements of the world (e.g., specific magic systems mentioned, unique technologies, important factions, distinct cultural aspects, important currencies or resources).
4.  The general speaking style or persona of the Narrator.

This summary will be used to help design thematically appropriate utility tools for this series.
Focus on information that would be useful for tailoring tools like inventories, stat trackers, quest logs, etc., to this specific world.
Your entire response MUST be this concise summary as plain text. Do not add any preamble or sign-off.
`
		// Renamed variable to match const
		summaryPromptData := struct {
			SeriesName                string
			NarratorName              string
			NarratorDescSnippet       string
			NarratorPersonalitySnippet string
			LorebookName              string
			LorebookDescSnippet       string
			LoreEntry1Comment         string
			LoreEntry1ContentSnippet  string
			LoreEntry2Comment         string
			LoreEntry2ContentSnippet  string
			LoreEntry3Comment         string
			LoreEntry3ContentSnippet  string
		}{
			SeriesName:                requestPayload.Series,
			NarratorName:              narratorCardOpt4.Data.Name,
			NarratorDescSnippet:       narratorDescSnippet,
			NarratorPersonalitySnippet: narratorPersSnippet,
			LorebookName:              masterLorebookOpt4.Name,
			LorebookDescSnippet:       loreDescSnippet,
			LoreEntry1Comment:         entryCommentSnippets[0],
			LoreEntry1ContentSnippet:  entrySnippets[0],
			LoreEntry2Comment:         entryCommentSnippets[1],
			LoreEntry2ContentSnippet:  entrySnippets[1],
			LoreEntry3Comment:         entryCommentSnippets[2],
			LoreEntry3ContentSnippet:  entrySnippets[2],
		}

		var filledSummaryPrompt strings.Builder
		// Using the const contextualSummaryPromptTemplate here
		summaryTmpl, err := template.New("contextualSummaryPrompt").Parse(contextualSummaryPromptTemplate)
		if err != nil {
			finalMessage += fmt.Sprintf("  ERROR parsing contextual summary prompt template: %v\\n", err)
			// Continue but subsequent steps might be less effective
		} else {
			if err := summaryTmpl.Execute(&filledSummaryPrompt, summaryPromptData); err != nil {
				finalMessage += fmt.Sprintf("  ERROR executing contextual summary prompt template: %v\n", err)
				// Continue
			}
		}
		
		worldContextSummary := "(Contextual summary generation failed or was skipped)" // Default if prompt fails
		if filledSummaryPrompt.Len() > 0 {
			summaryAIResponse, err := callGeminiAI(ctx, model, filledSummaryPrompt.String())
			if err != nil {
				finalMessage += fmt.Sprintf("  ERROR generating contextual summary (Ultimate Pack Step 3): %v\n", err)
				// Proceed with default summary, tool suggestions might be generic
			} else {
				worldContextSummary = strings.TrimSpace(summaryAIResponse)
				finalMessage += "  Successfully generated contextual summary.\n"
			}
		}
		finalMessage += "Step 3: Contextual summary generation attempt complete.\n\n"


		// --- Option 4 Step 4: AI-Driven Suggestion of 2 Utility Tools ---
		finalMessage += "Step 4: AI suggesting 2 tailored utility tools...\\n"
		log.Printf("Option 4 Step 4: AI suggesting tools for Series: %s, Log ID: %s", requestPayload.Series, logIdentifier)
		// Using a const for this multi-line string with backticks
		const toolSuggestionPromptTemplate = `
You are an expert game designer specializing in creating immersive user interface tools for fictional worlds.
The fictional series '{{.SeriesName}}' is characterized by the following (based on its Narrator and Lorebook):

--- WORLD & NARRATOR CONTEXTUAL SUMMARY ---
{{.WorldContextSummary}}
--- END WORLD & NARRATOR CONTEXTUAL SUMMARY ---

Based EXCLUSIVELY on the provided contextual summary, suggest EXACTLY TWO distinct types of SillyTavern UTILITY/TOOL character cards that would be:
a) Most thematically appropriate for '{{.SeriesName}}'.
b) Genuinely useful for a user interacting with the Narrator and exploring this Lorebook.
c) Complementary to each other (i.e., offer different kinds of utility).

Consider common utility needs like tracking player/character stats, managing inventory/currency, logging quests/events, referencing spells/abilities/tech, tracking faction reputation, managing party members, or consulting a bestiary/codex. Choose types that best fit the described world.

For EACH of the two suggested tools, provide:
1.  A "tool_type": A clear, descriptive label for the tool's function (e.g., "Character Status Tracker", "Chronicle of Deeds (Quest Log)", "Faction Allegiance Ledger", "Grimoire of Whispers (Spell Reference)", "Mercantile Satchel (Inventory & Currency)", "Bestiary of the Blighted Lands").
2.  A "tool_name": A creative and thematic name for the card itself that fits the style and specific elements of '{{.SeriesName}}' as described in the summary.
3.  A "tool_justification": A brief (1-2 sentences) explanation of *why* this specific tool (with this name and type) is particularly relevant and useful for THIS world/narrator.

Your response MUST be ONLY a single, valid JSON array containing exactly two objects. Each object must have "tool_type", "tool_name", and "tool_justification" keys.
Example JSON Array:
[
  {
    "tool_type": "Player Character Vitality & Resource Ledger",
    "tool_name": "The Emberheart Chronicle",
    "tool_justification": "This series features perilous combat and resource management. 'The Emberheart Chronicle' provides a thematic way to track a character's core stats and unique energy sources mentioned in the lore."
  },
  {
    "tool_type": "Registry of Pacts & Allegiances",
    "tool_name": "The Shadowbound Covenant",
    "tool_justification": "Given the focus on intricate faction politics and binding agreements, this tool will help users navigate their loyalties and the consequences of their choices within '{{.SeriesName}}'."
  }
]
Do NOT include any other text or explanation. Ensure the tool_names are unique and highly thematic to '{{.SeriesName}}'.
`
		// Renamed variable to match const
		suggestionPromptData := struct {
			SeriesName          string
			WorldContextSummary string
		}{
			SeriesName:          requestPayload.Series,
			WorldContextSummary: worldContextSummary,
		}
		var filledSuggestionPrompt strings.Builder
		// Using the const toolSuggestionPromptTemplate here
		suggestionTmpl, err := template.New("toolSuggestionPrompt").Parse(toolSuggestionPromptTemplate)
		if err != nil {
			finalMessage += fmt.Sprintf("  ERROR parsing tool suggestion prompt template: %v\\n", err)
		} else {
			if err := suggestionTmpl.Execute(&filledSuggestionPrompt, suggestionPromptData); err != nil {
				finalMessage += fmt.Sprintf("  ERROR executing tool suggestion prompt template: %v\n", err)
			}
		}
		
		var suggestedTools []AISuggestedTool
		if filledSuggestionPrompt.Len() > 0 {
			suggestionAIResponse, err := callGeminiAI(ctx, model, filledSuggestionPrompt.String())
			if err != nil {
				finalMessage += fmt.Sprintf("  ERROR getting AI tool suggestions (Ultimate Pack Step 4): %v\n", err)
			} else {
				if err := json.Unmarshal([]byte(suggestionAIResponse), &suggestedTools); err != nil {
					log.Printf("Failed to unmarshal AI tool suggestions (Ultimate Pack Step 4): %v. AI Response (Log ID %s): %s", err, logIdentifier, suggestionAIResponse)
					finalMessage += fmt.Sprintf("  ERROR parsing AI tool suggestions (Ultimate Pack Step 4). Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, suggestionAIResponse[:min(600, len(suggestionAIResponse))])
				} else if len(suggestedTools) != 2 {
					finalMessage += fmt.Sprintf("  AI did not suggest exactly two tools (Ultimate Pack Step 4). Received %d suggestions. Proceeding without tailored tools.\n", len(suggestedTools))
					suggestedTools = []AISuggestedTool{} // Clear if not exactly two
				} else {
					finalMessage += fmt.Sprintf("  Successfully received 2 AI tool suggestions: '%s' and '%s'.\n", suggestedTools[0].ToolName, suggestedTools[1].ToolName)
				}
			}
		} else {
			 finalMessage += "  Skipped AI tool suggestion due to template error.\n"
		}
		finalMessage += "Step 4: AI tool suggestion attempt complete.\n\n"


		// --- Option 4 Step 5: Generate Each Suggested Utility Card (up to 2) ---
		if len(suggestedTools) == 2 {
			for i, toolSuggestion := range suggestedTools {
				finalMessage += fmt.Sprintf("Step 5.%d: Generating tailored Utility Card: '%s' (%s)...\n", i+1, toolSuggestion.ToolName, toolSuggestion.ToolType)
				log.Printf("Option 4 Step 5.%d: Generating Utility Card: %s for Series: %s, Log ID: %s", i+1, toolSuggestion.ToolName, requestPayload.Series, logIdentifier)

				// Prepare context for this specific tool card
				// Example: Extracting some specific lore details for better tailoring.
				// This part needs to be robust and handle cases where masterLorebookOpt4 might be partially formed.
				var loreCurrencyName, loreStatsExamples, loreItemExamples, loreFactionExamples, loreMagicTechName string
				// Initialize with generic placeholders
				loreCurrencyName = "Standard Realm Currency (e.g., Gold Pieces, Credits)"
				loreStatsExamples = "Vitality, Essence, Might (refer to lorebook for specifics)"
				loreItemExamples = "Healing Draught, Mana Crystal (refer to lorebook for specifics)"
				loreFactionExamples = "(Refer to lorebook for specific faction names)"
				loreMagicTechName = "(Refer to lorebook for specific system names)"

				if len(masterLorebookOpt4.Entries) > 0 {
					// Attempt to extract more specific examples. This is still simplified.
					// A more sophisticated approach might involve tagging entries or more complex NLP.
					var foundCurrency, foundStats, foundItems, foundFactions, foundMagicTech bool
					for _, entry := range masterLorebookOpt4.Entries {
						lowerComment := strings.ToLower(entry.Comment)
						lowerContent := strings.ToLower(entry.Content)

						if !foundCurrency && (strings.Contains(lowerComment, "economy") || strings.Contains(lowerContent, "currency") || strings.Contains(lowerContent, "coin")) {
							loreCurrencyName = entry.Comment + " (e.g., " + extractExample(lowerContent, "currency") + ")"
							foundCurrency = true
						}
						if !foundStats && (strings.Contains(lowerComment, "character stat") || strings.Contains(lowerComment, "attribute")) {
							loreStatsExamples = entry.Comment + " (e.g., " + extractExample(lowerContent, "stat") + ")"
							foundStats = true
						}
						if !foundItems && (strings.Contains(lowerComment, "item") || strings.Contains(lowerComment, "artifact") || strings.Contains(lowerContent, "potion")) {
							loreItemExamples = entry.Comment + " (e.g., " + extractExample(lowerContent, "item") + ")"
							foundItems = true
						}
						if !foundFactions && (strings.Contains(lowerComment, "faction") || strings.Contains(lowerComment, "organization") || strings.Contains(lowerComment, "guild")) {
							loreFactionExamples = entry.Comment + " (e.g., " + extractExample(lowerContent, "faction") + ")"
							foundFactions = true
						}
						if !foundMagicTech && (strings.Contains(lowerComment, "magic system") || strings.Contains(lowerComment, "technology") || strings.Contains(lowerComment, "tech level")) {
							loreMagicTechName = entry.Comment + " (e.g., " + extractExample(lowerContent, "magic system") + ")"
							foundMagicTech = true
						}
						if foundCurrency && foundStats && foundItems && foundFactions && foundMagicTech {
							break // Found examples for all, no need to iterate further
						}
					}
				}

				// Construct the context injection string
				contextInjection := fmt.Sprintf(`
--- BEGIN CONTEXT INJECTION FOR THIS SPECIFIC TOOL ---
SERIES: %s
REQUESTED TOOL NAME: %s
REQUESTED TOOL TYPE: %s
TOOL JUSTIFICATION (Why this tool is relevant for %s): %s

NARRATOR CONTEXT (Emulate this style and incorporate relevant info):
Narrator's Name: %s
Narrator's Persona Snippet: %s 
Narrator's Speech Style Elements: (Refer to Narrator's full personality and speech pattern for nuanced style)

WORLD LORE CONTEXT (Incorporate these elements where appropriate):
Overall World Style & Key Themes: %s
Examples of In-World Terminology/Items/Currency (use these in your tool's data and examples, drawing from the broader lore):
  - Currency Name(s) (example): %s 
  - Common Measurable Stats/Attributes (example): %s
  - Example Items/Resources (example): %s
  - Key Factions (for reputation, etc.): %s
  - Magic System Name / Tech Level Name: %s
--- END CONTEXT INJECTION ---

Your primary goal is to make the '%s' feel like an authentic, indispensable artifact or interface from the world of '%s', fully consistent with the established Narrator and lore.
The 'description' field (initial data), 'first_mes', 'mes_example', and even the tool's 'personality' MUST reflect this deep integration.
				`, requestPayload.Series, toolSuggestion.ToolName, toolSuggestion.ToolType, requestPayload.Series, toolSuggestion.ToolJustification,
					narratorCardOpt4.Data.Name, narratorPersSnippet, /* narratorCardOpt4.Data.SpeechPattern - too long, rely on persona */
					worldContextSummary, // This contains style and themes
					loreCurrencyName,
					loreStatsExamples,
					loreItemExamples,
					loreFactionExamples,
					loreMagicTechName,
					toolSuggestion.ToolName, requestPayload.Series,
				)

				// The existing refined toolCardPromptTemplate from case "3"
				// We prepend the contextInjection to it.
				// toolCardPromptTemplate is now a global const and is a raw string literal.
				// The "\\n\\n" is fine here as it's a standard string literal being concatenated.
				finalToolCardPrompt := contextInjection + "\n\n" + toolCardPromptTemplate

				toolPromptData := struct { // Data for the base toolCardPromptTemplate part
					SeriesName  string
					ToolPurpose string // For the base template, this will be the AI suggested tool_name or tool_type
				}{
					SeriesName:  requestPayload.Series,
					ToolPurpose: toolSuggestion.ToolName, // Use the AI-suggested name as the "purpose" for the template
				}

				var filledFinalToolCardPrompt strings.Builder
				// We need to parse the combined prompt. It's okay if the base template uses {{.ToolPurpose}}
				// as we're filling that with the AI suggested name.
				finalToolTmpl, err := template.New(fmt.Sprintf("finalToolCardPrompt%d", i)).Parse(finalToolCardPrompt)
				if err != nil {
					finalMessage += fmt.Sprintf("  ERROR parsing final tailored tool card prompt template for '%s': %v\n", toolSuggestion.ToolName, err)
					continue // Skip this tool
				}

				if err := finalToolTmpl.Execute(&filledFinalToolCardPrompt, toolPromptData); err != nil {
					finalMessage += fmt.Sprintf("  ERROR executing final tailored tool card prompt template for '%s': %v\n", toolSuggestion.ToolName, err)
					continue // Skip this tool
				}

				actualFinalToolPrompt := filledFinalToolCardPrompt.String()
				
				utilityCardAIResponse, err := callGeminiAI(ctx, model, actualFinalToolPrompt)
				if err != nil {
					finalMessage += fmt.Sprintf("  ERROR generating tailored Utility Card '%s' (Ultimate Pack Step 5.%d): %v\n", toolSuggestion.ToolName, i+1, err)
					continue // Skip this tool if generation fails
				}

				var utilityCard CharacterCardV2
				if err := json.Unmarshal([]byte(utilityCardAIResponse), &utilityCard); err != nil {
					log.Printf("Failed to unmarshal tailored Utility Card '%s': %v. AI Response (Log ID %s): %s", toolSuggestion.ToolName, err, logIdentifier, utilityCardAIResponse)
					finalMessage += fmt.Sprintf("  ERROR parsing AI response for tailored Utility Card '%s'. Raw AI output (Log ID %s): %s[:600]...\n", toolSuggestion.ToolName, logIdentifier, utilityCardAIResponse[:min(600, len(utilityCardAIResponse))])
					continue
				}
				if utilityCard.Spec == "" { utilityCard.Spec = "chara_card_v2" }
				if utilityCard.SpecVersion == "" { utilityCard.SpecVersion = "2.0" }
				if utilityCard.Data.Name == "" { utilityCard.Data.Name = toolSuggestion.ToolName } // Use AI suggested name
				utilityCard.Data.CharacterBook = nil

				utilityJsonData, _ := json.MarshalIndent(utilityCard, "", "  ")
				allGeneratedJSONsOpt4 = append(allGeneratedJSONsOpt4, string(utilityJsonData))
				utilityFilePath, saveErr := saveJSONToFile(requestPayload.Series, fmt.Sprintf("tailored_tool_%d", i+1), utilityCard.Data.Name, logIdentifier, utilityJsonData)
				if saveErr != nil {
					finalMessage += fmt.Sprintf("  Successfully generated tailored Utility Card '%s', but FAILED to save. Error: %s\n", utilityCard.Data.Name, saveErr.Error())
				} else {
					finalMessage += fmt.Sprintf("  Successfully generated and saved tailored Utility Card '%s' to: %s\n", utilityCard.Data.Name, utilityFilePath)
				}
			}
		} else if filledSuggestionPrompt.Len() > 0 { // Only note if suggestions were attempted but failed parsing or count
			 finalMessage += "  Skipped generation of tailored utility tools as AI suggestions were not successfully processed.\n"
		}
		finalMessage += "Step 5: Tailored utility card generation attempts complete.\n\n"

		generatedJSONString = strings.Join(allGeneratedJSONsOpt4, "\n\n"+CHARACTER_CARD_SEPARATOR+"\n\n")
		finalMessage += fmt.Sprintf("Option 4: ULTIMATE PACK for '%s' processing finished. Check all generated files and messages.\n", requestPayload.Series)


	default:
		sendJSONError(w, "Invalid option selected.", http.StatusBadRequest, logIdentifier)
		return
	}

	// Prepare and send the final successful response
	finalResponse := ResponsePayload{
		Series:           requestPayload.Series,
		OptionChosen:     optionText,
		ModelUsed:        requestPayload.Model,
		APIKeyReceived:   true,                // If we reach here, key was non-empty and client initialized
		Message:          finalMessage,        // Contains step-by-step progress and file paths
		GeneratedContent: generatedJSONString, // Contains all JSONs for the option, separated if multiple
		Timestamp:        time.Now().Format(time.RFC3339),
		LogIdentifier:    logIdentifier, // Crucial for user to find their files
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(finalResponse); err != nil {
		// This error is mostly server-side if the headers and status are already sent.
		log.Printf("Error encoding final success response: %v", err)
	}
}

// main function to start the HTTP server.
func main() {
	// Standard library text/template is used for prompt templating.
	// No external dependencies beyond what's in go.mod are introduced for this feature.

	// Serve static files (like index.html) from the current directory where the executable is run.
	// This assumes index.html is in the same directory as the compiled Go application.
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	// Handle the /generate endpoint for AI content generation requests
	http.HandleFunc("/generate", generateContentHandler)

	port := "8080" // Define the port the server will listen on
	log.Printf("AI Fiction Forge server starting on port %s", port)
	log.Printf("Access the UI via http://localhost:%s in your browser.", port)
	log.Printf("Ensure index.html is in the same directory as the executable.")
	log.Printf("Generated JSONs will be saved in a './jsons/' subdirectory, organized by series and timestamp.")
	// Start the server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Helper function to find the minimum of two integers.
// It\'s good practice to have such small utility functions.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractExample is a placeholder for a more sophisticated function to get relevant examples from text.
// For now, it returns a generic string or a snippet.
func extractExample(content string, category string) string {
	// This is a very basic placeholder. A real implementation might use regex or NLP.
	// For now, just return a generic indication or a snippet.
	if len(content) > 30 {
		return content[:30] + "..."
	}
	return content
}
