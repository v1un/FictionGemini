package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"

	// No longer need "github.com/google/generative-ai-go/genai" directly here
	"workspace/FictionGeminiRewritten/internal/ai"
	"workspace/FictionGeminiRewritten/internal/models"
	"workspace/FictionGeminiRewritten/internal/prompts"
	"workspace/FictionGeminiRewritten/internal/util"
)

const (
	baseJSONSaveDir = "./jsons"
)

// OrchestratorService handles the core logic of generating content based on options.
type OrchestratorService struct {
	// geminiClient removed
}

// NewOrchestratorService creates a new OrchestratorService.
func NewOrchestratorService() *OrchestratorService { // geminiClient parameter removed
	return &OrchestratorService{}
}

// ProcessGenerationRequest orchestrates the content generation based on the request payload.
// It returns the final generated JSON string (can be multiple, concatenated), a detailed message log,
// the chosen option text for the response, and an error if something went critically wrong.
func (s *OrchestratorService) ProcessGenerationRequest(
	ctx context.Context,
	payload models.RequestPayload,
	logIdentifier string,
	apiKey string, // Added apiKey
) (generatedJSONString string, messageLog string, optionText string, err error) {

	// model := s.geminiClient.GenerativeModel(payload.Model) // Removed
	// model.GenerationConfig.ResponseMIMEType = "application/json" // As in original, but commented out.

	var messages []string // To accumulate log messages for the user

	switch payload.Option {
	case "1":
		optionText = "Lorebook Only (Comprehensive)"
		messages = append(messages, fmt.Sprintf("Processing Option 1: Comprehensive Lorebook for '%s'.\n", payload.Series))
		
		promptString := fmt.Sprintf(prompts.ComprehensiveLorebookPrompt,
			payload.Series, payload.Series, payload.Series, payload.Series, payload.Series, payload.Series)

		// Call AI (updated)
		aiResponse, aiErr := ai.CallGeminiAPI(ctx, apiKey, payload.Model, promptString)
		if aiErr != nil {
			messages = append(messages, fmt.Sprintf("  ERROR generating Comprehensive Lorebook: %v\n", aiErr))
			return "", strings.Join(messages, ""), optionText, fmt.Errorf("AI generation failed for Comprehensive Lorebook: %w", aiErr)
		}

		var loreBook models.Lorebook
		if err := json.Unmarshal([]byte(aiResponse), &loreBook); err != nil {
			log.Printf("Failed to unmarshal Comprehensive Lorebook (Log ID %s): %v. AI Response: %s", logIdentifier, err, aiResponse)
			messages = append(messages, fmt.Sprintf("  ERROR parsing AI response for Comprehensive Lorebook. Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, aiResponse[:util.Min(600, len(aiResponse))]))
			return "", strings.Join(messages, ""), optionText, fmt.Errorf("failed to parse AI response for Comprehensive Lorebook: %w", err)
		}

		loreBook.Enabled = true
		if loreBook.Name == "" {
			loreBook.Name = fmt.Sprintf("Comprehensive Lore for %s", payload.Series)
		}
		for i := range loreBook.Entries {
			loreBook.Entries[i].Enabled = true
		}

		jsonData, _ := json.MarshalIndent(loreBook, "", "  ")
		generatedJSONString = string(jsonData)

		filePath, saveErr := SaveJSONToFile(baseJSONSaveDir, payload.Series, "lorebook_comprehensive", loreBook.Name, logIdentifier, jsonData)
		if saveErr != nil {
			messages = append(messages, fmt.Sprintf("  Successfully generated Comprehensive Lorebook JSON, but FAILED to save to server file system. Error: %s\n", saveErr.Error()))
			log.Printf("Failed to save Comprehensive Lorebook JSON to file (Log ID %s): %v", logIdentifier, saveErr)
		} else {
			messages = append(messages, fmt.Sprintf("  Successfully generated Comprehensive Lorebook JSON and saved to: %s\n", filePath))
		}
		messages = append(messages, "Comprehensive Lorebook generation complete.\n")
		messageLog = strings.Join(messages, "")
		return generatedJSONString, messageLog, optionText, nil

	case "3": // Utility/Tool Card
		optionText = fmt.Sprintf("Utility/Tool Card Creator (%s)", payload.ToolCardPurpose)
		messages = append(messages, fmt.Sprintf("Processing Option 3: Utility/Tool Card ('%s') for series '%s'.\n", payload.ToolCardPurpose, payload.Series))

		if strings.TrimSpace(payload.ToolCardPurpose) == "" {
			messages = append(messages, "  ERROR: Tool Card Purpose is missing for Option 3.\n")
			return "", strings.Join(messages, ""), optionText, fmt.Errorf("missing Tool Card Purpose for Option 3")
		}
		
		promptData := struct {
			SeriesName  string
			ToolPurpose string
		}{
			SeriesName:  payload.Series,
			ToolPurpose: payload.ToolCardPurpose,
		}

		var filledPrompt bytes.Buffer
		tmpl, err := template.New("toolCardPrompt").Parse(prompts.ToolCardPromptTemplate)
		if err != nil {
			messages = append(messages, fmt.Sprintf("  ERROR: Failed to parse tool card prompt template: %v\n", err))
			return "", strings.Join(messages, ""), optionText, fmt.Errorf("failed to parse tool card prompt template: %w", err)
		}
		if err := tmpl.Execute(&filledPrompt, promptData); err != nil {
			messages = append(messages, fmt.Sprintf("  ERROR: Failed to execute tool card prompt template: %v\n", err))
			return "", strings.Join(messages, ""), optionText, fmt.Errorf("failed to execute tool card prompt template: %w", err)
		}
		actualPrompt := filledPrompt.String()

		// Call AI (updated)
		aiResponse, aiErr := ai.CallGeminiAPI(ctx, apiKey, payload.Model, actualPrompt)
		if aiErr != nil {
			messages = append(messages, fmt.Sprintf("  ERROR generating Tool Card ('%s'): %v\n", payload.ToolCardPurpose, aiErr))
			return "", strings.Join(messages, ""), optionText, fmt.Errorf("AI generation failed for Tool Card ('%s'): %w", payload.ToolCardPurpose, aiErr)
		}

		var toolCard models.CharacterCardV2
		if err := json.Unmarshal([]byte(aiResponse), &toolCard); err != nil {
			log.Printf("Failed to unmarshal Tool Card (Log ID %s): %v. AI Response: %s", logIdentifier, err, aiResponse)
			messages = append(messages, fmt.Sprintf("  ERROR parsing AI response for Tool Card ('%s'). Raw AI output (check logs for ID %s for details): %s\n", payload.ToolCardPurpose, logIdentifier, aiResponse[:util.Min(600, len(aiResponse))]))
			return "", strings.Join(messages, ""), optionText, fmt.Errorf("failed to parse AI response for Tool Card ('%s'): %w", payload.ToolCardPurpose, err)
		}

		if toolCard.Spec == "" {toolCard.Spec = "chara_card_v2"}
		if toolCard.SpecVersion == "" {toolCard.SpecVersion = "2.0"}
		if toolCard.Data.Name == "" {
			toolCard.Data.Name = fmt.Sprintf("%s for %s", payload.ToolCardPurpose, payload.Series)
		}
		toolCard.Data.CharacterBook = nil 

		jsonData, _ := json.MarshalIndent(toolCard, "", "  ")
		generatedJSONString = string(jsonData)

		filePath, saveErr := SaveJSONToFile(baseJSONSaveDir, payload.Series, "tool_card", toolCard.Data.Name, logIdentifier, jsonData)
		if saveErr != nil {
			messages = append(messages, fmt.Sprintf("  Successfully generated Tool Card ('%s'), but FAILED to save. Error: %s\n", payload.ToolCardPurpose, saveErr.Error()))
		} else {
			messages = append(messages, fmt.Sprintf("  Successfully generated and saved Tool Card ('%s') to: %s\n", payload.ToolCardPurpose, filePath))
		}
		messages = append(messages, fmt.Sprintf("Option 3: Utility/Tool Card ('%s') generation complete.\n", payload.ToolCardPurpose))
		messageLog = strings.Join(messages, "")
		return generatedJSONString, messageLog, optionText, nil

	case "2": // Narrator Card + Master Lorebook
		optionText = "Narrator Card + Master Lorebook (Refined)"
		messages = append(messages, fmt.Sprintf("Processing Option 2: Narrator Card + Master Lorebook for '%s'. This is a multi-step process.\n\n", payload.Series))
		var allGeneratedJSONsOpt2 []string

		// Step 1: Generate Narrator Character Card (updated call)
		_, narratorJSON, errNarrator := s.generateNarratorCard(ctx, apiKey, payload.Model, payload.Series, logIdentifier, &messages)
		if errNarrator != nil {
			return "", strings.Join(messages, ""), optionText, errNarrator
		}
		allGeneratedJSONsOpt2 = append(allGeneratedJSONsOpt2, narratorJSON)

		// Step 2: Generate Master Lorebook (updated call)
		_, lorebookJSON, errLorebook := s.generateMasterLorebook(ctx, apiKey, payload.Model, payload.Series, logIdentifier, &messages)
		if errLorebook != nil {
			log.Printf("Error in Option 2, Step 2 (Master Lorebook) but Narrator Card might be okay. Log ID: %s, Err: %v", logIdentifier, errLorebook)
		} else {
			allGeneratedJSONsOpt2 = append(allGeneratedJSONsOpt2, lorebookJSON)
		}

		generatedJSONString = strings.Join(allGeneratedJSONsOpt2, "\n\n"+models.CHARACTER_CARD_SEPARATOR+"\n\n")
		messages = append(messages, "Option 2 (Narrator Card + Master Lorebook) processing finished.\n")
		messageLog = strings.Join(messages, "")
		return generatedJSONString, messageLog, optionText, nil 

	case "4": // Ultimate Pack (Narrator + Lorebook + 2 AI-Suggested Tools)
		optionText = "Narrator + Lorebook + Tailored Utils (Ultimate Pack)"
		messages = append(messages, fmt.Sprintf("Processing Option 4: ULTIMATE PACK for '%s'. This is a multi-step process and will take time.\n\n", payload.Series))
		var allGeneratedJSONsOpt4 []string
		var errOption4 error 

		// Step 1: Generate Narrator Character Card (updated call)
		narratorCard, narratorJSON, errNarrator := s.generateNarratorCard(ctx, apiKey, payload.Model, payload.Series, logIdentifier, &messages)
		if errNarrator != nil {
			errOption4 = errNarrator
			return "", strings.Join(messages, ""), optionText, errOption4
		}
		allGeneratedJSONsOpt4 = append(allGeneratedJSONsOpt4, narratorJSON)

		// Step 2: Generate Master Lorebook (updated call)
		masterLorebook, lorebookJSON, errLorebook := s.generateMasterLorebook(ctx, apiKey, payload.Model, payload.Series, logIdentifier, &messages)
		if errLorebook != nil {
			log.Printf("Error in Option 4, Step 2 (Master Lorebook) but continuing. Log ID: %s, Err: %v", logIdentifier, errLorebook)
		} else {
			allGeneratedJSONsOpt4 = append(allGeneratedJSONsOpt4, lorebookJSON)
		}

		// Step 3: Generate Contextual Summary (updated call)
		worldContextSummary, _ := s.generateContextualSummary(ctx, apiKey, payload.Model, payload.Series, narratorCard.Data, masterLorebook, logIdentifier, &messages)
		
		// Step 4: AI Suggest Utility Tools (updated call)
		suggestedTools, errSuggest := s.suggestUtilityTools(ctx, apiKey, payload.Model, payload.Series, worldContextSummary, logIdentifier, &messages)
		if errSuggest != nil {
			log.Printf("Error in Option 4, Step 4 (Suggest Tools) but continuing. Log ID: %s, Err: %v", logIdentifier, errSuggest)
			suggestedTools = []models.AISuggestedTool{} 
		}

		// Step 5: Generate Each Suggested Utility Card (if suggestions were successful)
		if len(suggestedTools) == 2 {
			for i, toolSuggestion := range suggestedTools {
				// Updated call
				utilityJSON, errTool := s.generateTailoredUtilityCard(ctx, apiKey, payload.Model, payload.Series, toolSuggestion, narratorCard.Data, masterLorebook, worldContextSummary, logIdentifier, i, &messages)
				if errTool != nil {
					log.Printf("Error in Option 4, Step 5 (Generate Tool %d: %s) but continuing. Log ID: %s, Err: %v", i+1, toolSuggestion.ToolName, logIdentifier, errTool)
				} else {
					allGeneratedJSONsOpt4 = append(allGeneratedJSONsOpt4, utilityJSON)
				}
			}
			messages = append(messages, "Tailored utility card generation attempts complete.\n\n")
		} else if errSuggest == nil {
			messages = append(messages, "Skipped generation of tailored utility tools as AI suggestions were not successfully processed (wrong count).\n\n")
		}

		generatedJSONString = strings.Join(allGeneratedJSONsOpt4, "\n\n"+models.CHARACTER_CARD_SEPARATOR+"\n\n")
		messages = append(messages, fmt.Sprintf("Option 4: ULTIMATE PACK for '%s' processing finished. Check all generated files and messages.\n", payload.Series))
		messageLog = strings.Join(messages, "")
		return generatedJSONString, messageLog, optionText, nil 

	default:
		return "", "Invalid option selected in orchestrator.", "Unknown Option", fmt.Errorf("invalid option: %s", payload.Option)
	}
}

// Helper function to execute a text/template (no changes needed here)
func executeTemplate(templateName string, templateStr string, data interface{}) (string, error) {
	var filledPrompt bytes.Buffer
	tmpl, err := template.New(templateName).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template '%s': %w", templateName, err)
	}
	if err := tmpl.Execute(&filledPrompt, data); err != nil {
		return "", fmt.Errorf("failed to execute template '%s': %w", templateName, err)
	}
	return filledPrompt.String(), nil
}


// --- Helper functions for multi-step generation processes ---

// generateNarratorCard (signature and AI call updated)
func (s *OrchestratorService) generateNarratorCard(ctx context.Context, apiKey string, modelName string, seriesName, logIdentifier string, currentMessages *[]string) (models.CharacterCardV2, string, error) {
	*currentMessages = append(*currentMessages, "Step: Generating highly detailed Narrator Character Card...\n")
	narratorName := fmt.Sprintf("The Narrator of %s", seriesName)

	promptStr := fmt.Sprintf(prompts.NarratorCardPrompt,
		seriesName, seriesName, narratorName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, 
		seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, 
		seriesName, seriesName, seriesName)

	aiResponse, err := ai.CallGeminiAPI(ctx, apiKey, modelName, promptStr) // Updated call
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR generating Narrator Card: %v\n", err))
		return models.CharacterCardV2{}, "", fmt.Errorf("AI generation failed for Narrator Card: %w", err)
	}

	var card models.CharacterCardV2
	if err := json.Unmarshal([]byte(aiResponse), &card); err != nil {
		log.Printf("Failed to unmarshal Narrator Card (Log ID %s): %v. AI Response: %s", logIdentifier, err, aiResponse)
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR parsing Narrator Card. Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, aiResponse[:util.Min(600, len(aiResponse))]))
		return models.CharacterCardV2{}, "", fmt.Errorf("failed to parse AI response for Narrator Card: %w", err)
	}

	if card.Spec == "" {card.Spec = "chara_card_v2"}
	if card.SpecVersion == "" {card.SpecVersion = "2.0"}
	if card.Data.Name == "" {card.Data.Name = narratorName}
	card.Data.CharacterBook = nil 

	jsonData, _ := json.MarshalIndent(card, "", "  ")
	jsonStr := string(jsonData)

	filePath, saveErr := SaveJSONToFile(baseJSONSaveDir, seriesName, "narrator_card", card.Data.Name, logIdentifier, jsonData)
	if saveErr != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  Successfully generated Narrator Card JSON, but FAILED to save. Error: %s\n", saveErr.Error()))
	} else {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  Successfully generated and saved Narrator Card to: %s\n", filePath))
	}
	*currentMessages = append(*currentMessages, "Narrator Card generation complete.\n\n")
	return card, jsonStr, nil
}

// generateMasterLorebook (signature and AI call updated)
func (s *OrchestratorService) generateMasterLorebook(ctx context.Context, apiKey string, modelName string, seriesName, logIdentifier string, currentMessages *[]string) (models.Lorebook, string, error) {
	*currentMessages = append(*currentMessages, "Step: Generating Master Lorebook (Refined)...\n")
	
	promptStr := fmt.Sprintf(prompts.MasterLorebookPrompt,
		seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName, seriesName)

	aiResponse, err := ai.CallGeminiAPI(ctx, apiKey, modelName, promptStr) // Updated call
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR generating Master Lorebook: %v\n", err))
		return models.Lorebook{}, "", fmt.Errorf("AI generation failed for Master Lorebook: %w", err)
	}

	var lorebook models.Lorebook
	if err := json.Unmarshal([]byte(aiResponse), &lorebook); err != nil {
		log.Printf("Failed to unmarshal Master Lorebook (Log ID %s): %v. AI Response: %s", logIdentifier, err, aiResponse)
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR parsing Master Lorebook. Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, aiResponse[:util.Min(600, len(aiResponse))]))
		return models.Lorebook{}, "", fmt.Errorf("failed to parse AI response for Master Lorebook: %w", err)
	}

	lorebook.Enabled = true
	if lorebook.Name == "" { lorebook.Name = fmt.Sprintf("Master Lorebook for %s", seriesName) }
	for i := range lorebook.Entries {
		lorebook.Entries[i].Enabled = true
	}

	jsonData, _ := json.MarshalIndent(lorebook, "", "  ")
	jsonStr := string(jsonData)

	filePath, saveErr := SaveJSONToFile(baseJSONSaveDir, seriesName, "master_lorebook", lorebook.Name, logIdentifier, jsonData)
	if saveErr != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  Successfully generated Master Lorebook JSON, but FAILED to save. Error: %s\n", saveErr.Error()))
	} else {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  Successfully generated and saved Master Lorebook to: %s\n", filePath))
	}
	*currentMessages = append(*currentMessages, "Master Lorebook generation complete.\n\n")
	return lorebook, jsonStr, nil
}

// generateContextualSummary (signature and AI call updated)
func (s *OrchestratorService) generateContextualSummary(ctx context.Context, apiKey string, modelName string, seriesName string, narratorData models.CardData, lorebookData models.Lorebook, logIdentifier string, currentMessages *[]string) (string, error) {
	*currentMessages = append(*currentMessages, "Step: Generating Contextual Summary for AI Tool Suggestion...\n")

	narratorJSON, _ := json.MarshalIndent(narratorData, "", "  ")
	lorebookJSON, _ := json.MarshalIndent(lorebookData, "", "  ")

	promptData := struct {
		SeriesName   string
		NarratorJSON string
		LorebookJSON string
	}{
		SeriesName:   seriesName,
		NarratorJSON: string(narratorJSON),
		LorebookJSON: string(lorebookJSON),
	}
	
	actualPrompt, err := executeTemplate("contextSummaryPrompt", prompts.ContextualSummaryPrompt, promptData)
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR preparing prompt for Contextual Summary: %v\n", err))
		return "", err // Return error as this step is crucial for next
	}

	aiResponse, err := ai.CallGeminiAPI(ctx, apiKey, modelName, actualPrompt) // Updated call
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR generating Contextual Summary: %v\n", err))
		return "", err // Return error
	}
	
	// The response is expected to be a plain text summary.
	// No unmarshalling, just return the string.
	// Basic validation: ensure it's not empty.
	if strings.TrimSpace(aiResponse) == "" {
		*currentMessages = append(*currentMessages, "  WARNING: AI returned an empty Contextual Summary.\n")
		return "", fmt.Errorf("AI returned an empty contextual summary")
	}

	*currentMessages = append(*currentMessages, "Contextual Summary generated.\n\n")
	return aiResponse, nil
}

// suggestUtilityTools (signature and AI call updated)
func (s *OrchestratorService) suggestUtilityTools(ctx context.Context, apiKey string, modelName string, seriesName string, worldContextSummary string, logIdentifier string, currentMessages *[]string) ([]models.AISuggestedTool, error) {
	*currentMessages = append(*currentMessages, "Step: AI Suggesting 2 Tailored Utility Tools...\n")

	if strings.TrimSpace(worldContextSummary) == "" {
		*currentMessages = append(*currentMessages, "  Skipping AI tool suggestion: World Context Summary is empty.\n")
		return nil, fmt.Errorf("world context summary is empty, cannot suggest tools")
	}
	
	promptData := struct {
		SeriesName          string
		WorldContextSummary string
	}{
		SeriesName:          seriesName,
		WorldContextSummary: worldContextSummary,
	}

	actualPrompt, err := executeTemplate("toolSuggestionPrompt", prompts.ToolSuggestionPrompt, promptData)
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR preparing prompt for Tool Suggestion: %v\n", err))
		return nil, err
	}

	aiResponse, err := ai.CallGeminiAPI(ctx, apiKey, modelName, actualPrompt) // Updated call
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR from AI during Tool Suggestion: %v\n", err))
		return nil, err
	}

	var suggestions struct {
		Tools []models.AISuggestedTool `json:"suggested_tools"`
	}
	if err := json.Unmarshal([]byte(aiResponse), &suggestions); err != nil {
		log.Printf("Failed to unmarshal AI Tool Suggestions (Log ID %s): %v. AI Response: %s", logIdentifier, err, aiResponse)
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR parsing AI Tool Suggestions. Raw AI output (check logs for ID %s for details): %s\n", logIdentifier, aiResponse[:util.Min(600, len(aiResponse))]))
		return nil, fmt.Errorf("failed to parse AI response for tool suggestions: %w", err)
	}

	if len(suggestions.Tools) != 2 {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  WARNING: AI suggested %d tools instead of 2. Proceeding with what was given, but this might impact utility card generation.\n", len(suggestions.Tools)))
		// Depending on strictness, could return an error here. For now, allow it but log.
	}
	
	if len(suggestions.Tools) > 0 {
		for i, tool := range suggestions.Tools {
			*currentMessages = append(*currentMessages, fmt.Sprintf("  AI Suggested Tool %d: Type='%s', Name='%s', Justification='%s'\n", i+1, tool.ToolType, tool.ToolName, tool.ToolJustification))
		}
	} else {
		*currentMessages = append(*currentMessages, "  AI did not suggest any tools.\n")
	}
	*currentMessages = append(*currentMessages, "AI Tool Suggestion phase complete.\n\n")
	return suggestions.Tools, nil
}

// generateTailoredUtilityCard (signature and AI call updated)
func (s *OrchestratorService) generateTailoredUtilityCard(
	ctx context.Context, apiKey string, modelName string, seriesName string,
	toolSuggestion models.AISuggestedTool,
	narratorData models.CardData, // Used for context
	lorebookData models.Lorebook, // Used for context
	worldContextSummary string, // Used for context
	logIdentifier string, toolIndex int, currentMessages *[]string,
) (string, error) {
	*currentMessages = append(*currentMessages, fmt.Sprintf("Step: Generating Tailored Utility Card %d: '%s' (Type: '%s')...\n", toolIndex+1, toolSuggestion.ToolName, toolSuggestion.ToolType))

	narratorJSON, _ := json.MarshalIndent(narratorData, "", "  ")
	lorebookJSON, _ := json.MarshalIndent(lorebookData, "", "  ")

	promptData := struct {
		SeriesName          string
		ToolName            string
		ToolType            string
		ToolJustification   string
		NarratorCardJSON    string
		MasterLorebookJSON  string
		WorldContextSummary string
	}{
		SeriesName:          seriesName,
		ToolName:            toolSuggestion.ToolName,
		ToolType:            toolSuggestion.ToolType,
		ToolJustification:   toolSuggestion.ToolJustification,
		NarratorCardJSON:    string(narratorJSON),
		MasterLorebookJSON:  string(lorebookJSON),
		WorldContextSummary: worldContextSummary,
	}

	actualPrompt, err := executeTemplate("tailoredToolCardPrompt", prompts.ToolCardPromptTemplate, promptData)
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR preparing prompt for Tailored Utility Card '%s': %v\n", toolSuggestion.ToolName, err))
		return "", err
	}
	
	aiResponse, err := ai.CallGeminiAPI(ctx, apiKey, modelName, actualPrompt) // Updated call
	if err != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR from AI generating Tailored Utility Card '%s': %v\n", toolSuggestion.ToolName, err))
		return "", err
	}

	var card models.CharacterCardV2
	if err := json.Unmarshal([]byte(aiResponse), &card); err != nil {
		log.Printf("Failed to unmarshal Tailored Utility Card '%s' (Log ID %s): %v. AI Response: %s", toolSuggestion.ToolName, logIdentifier, err, aiResponse)
		*currentMessages = append(*currentMessages, fmt.Sprintf("  ERROR parsing Tailored Utility Card '%s'. Raw AI output (check logs for ID %s for details): %s\n", toolSuggestion.ToolName, logIdentifier, aiResponse[:util.Min(600, len(aiResponse))]))
		return "", fmt.Errorf("failed to parse AI response for tailored utility card '%s': %w", toolSuggestion.ToolName, err)
	}

	// Validate/Defaults
	if card.Spec == "" {card.Spec = "chara_card_v2"}
	if card.SpecVersion == "" {card.SpecVersion = "2.0"}
	if card.Data.Name == "" { card.Data.Name = toolSuggestion.ToolName } // Default to suggested name
	card.Data.CharacterBook = nil // No embedded lorebook for utility cards

	jsonData, _ := json.MarshalIndent(card, "", "  ")
	jsonStr := string(jsonData)

	fileName := fmt.Sprintf("utility_card_ai_suggested_%d", toolIndex+1)
	filePath, saveErr := SaveJSONToFile(baseJSONSaveDir, seriesName, fileName, card.Data.Name, logIdentifier, jsonData)
	if saveErr != nil {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  Successfully generated Tailored Utility Card '%s' JSON, but FAILED to save. Error: %s\n", toolSuggestion.ToolName, saveErr.Error()))
	} else {
		*currentMessages = append(*currentMessages, fmt.Sprintf("  Successfully generated and saved Tailored Utility Card '%s' to: %s\n", toolSuggestion.ToolName, filePath))
	}
	*currentMessages = append(*currentMessages, fmt.Sprintf("Tailored Utility Card '%s' generation complete.\n\n", toolSuggestion.ToolName))
	return jsonStr, nil
}
