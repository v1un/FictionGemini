import React, { useState, useEffect } from 'react';
import './App.css'; // Assuming App.css has been copied

const CHARACTER_CARD_SEPARATOR = "CHARACTER_CARD_SEPARATOR_AI_FICTION_FORGE"; // Keep this for now
const API_KEY_STORAGE_KEY = 'geminiApiKey';

function App() {
  const [apiKey, setApiKey] = useState('');
  const [series, setSeries] = useState('');
  const [model, setModel] = useState('gemini-1.5-flash-latest'); // Default model
  const [option, setOption] = useState('1'); // Corresponds to prompt_type
  const [toolCardPurpose, setToolCardPurpose] = useState('');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [responseData, setResponseData] = useState(null);
  const [generatedContentParts, setGeneratedContentParts] = useState([]);

  // Load API key from localStorage on initial render
  useEffect(() => {
    const storedApiKey = localStorage.getItem(API_KEY_STORAGE_KEY);
    if (storedApiKey) {
      setApiKey(storedApiKey);
    }
  }, []);

  // Save API key to localStorage when it changes
  useEffect(() => {
    if (apiKey) { // Only save if apiKey is not empty, to allow clearing
        localStorage.setItem(API_KEY_STORAGE_KEY, apiKey);
    } else {
        localStorage.removeItem(API_KEY_STORAGE_KEY); // Remove if cleared
    }
  }, [apiKey]);


  useEffect(() => {
    if (option !== '3') {
      setToolCardPurpose('');
    }
  }, [option]);

  const mapOptionToPromptType = (opt) => {
    switch (opt) {
      case '1': return 'lorebook_only';
      case '2': return 'narrator_and_lorebook';
      case '3': return 'tool_card';
      case '4': return 'ultimate_pack';
      default: return '';
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setResponseData(null);
    setGeneratedContentParts([]);

    if (!apiKey.trim()) {
      setError("API Key is required.");
      setLoading(false);
      return;
    }
    if (!series.trim()) {
      setError("Series Name is required.");
      setLoading(false);
      return;
    }
    if (!model.trim()) {
      setError("AI Model is required.");
      setLoading(false);
      return;
    }

    const promptType = mapOptionToPromptType(option);
    if (!promptType) {
        setError("Invalid generation option selected.");
        setLoading(false);
        return;
    }

    const payload = {
      api_key: apiKey,
      story_state: series,
      model_name: model,
      prompt_type: promptType,
      user_choice: option, // Send original option as user_choice for logging
      custom_prompt: '', // Initialize custom_prompt
    };

    if (option === '3') {
      if (!toolCardPurpose.trim()) {
        setError("Tool Card Purpose is required for Option 3.");
        setLoading(false);
        return;
      }
      payload.custom_prompt = toolCardPurpose;
    }

    try {
      const response = await fetch('http://localhost:51605/generate', { // Updated URL
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || `Server responded with ${response.status}: ${response.statusText}`);
      }
      
      setResponseData(data); // This now contains { story_segment, message, log_identifier, etc. }
      if (data.story_segment) {
        // Assuming story_segment might contain multiple parts separated by the same separator
        setGeneratedContentParts(data.story_segment.split(CHARACTER_CARD_SEPARATOR));
      } else if (!data.error) {
        // Handle cases where there's no primary story_segment but also no explicit error (e.g., just a message)
      }

    } catch (err) {
      setError(err.message);
      console.error("Fetch error:", err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>AI Fiction Forge (Rewritten Frontend)</h1>
        <p>Generate SillyTavern Character Cards & Lorebooks with AI</p>
      </header>

      <div className="form-container">
        <h2>Generation Parameters</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="apiKey">Gemini API Key:</label>
            <input
              type="password"
              id="apiKey"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="Enter your Gemini API Key (stored in localStorage)"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="series">Series Name (Story State):</label>
            <input
              type="text"
              id="series"
              value={series}
              onChange={(e) => setSeries(e.target.value)}
              placeholder="e.g., The Eldoria Chronicles"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="model">AI Model Name:</label>
            <input
              type="text"
              id="model"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              placeholder="e.g., gemini-1.5-flash-latest"
              required
            />
             <small>Example: gemini-1.5-flash-latest, gemini-pro. Ensure the model supports the API key.</small>
          </div>

          <div className="form-group">
            <label htmlFor="option">Generation Option:</label>
            <select id="option" value={option} onChange={(e) => setOption(e.target.value)}>
              <option value="1">1. Lorebook Only (Comprehensive)</option>
              <option value="2">2. Narrator Card + Master Lorebook (Refined)</option>
              <option value="3">3. Utility/Tool Card (User-Defined Purpose)</option>
              <option value="4">4. Ultimate Pack: Narrator + Lorebook + 2 AI-Suggested Utility Cards</option>
            </select>
          </div>

          {option === '3' && (
            <div className="form-group">
              <label htmlFor="toolCardPurpose">Tool Card Purpose (Custom Prompt for Option 3):</label>
              <input
                type="text"
                id="toolCardPurpose"
                value={toolCardPurpose}
                onChange={(e) => setToolCardPurpose(e.target.value)}
                placeholder="e.g., Player Character Stats, Party Inventory"
                required={option === '3'}
              />
              <small>Describe what the utility card should do. Required only if Option 3 is selected.</small>
            </div>
          )}
          
          <button type="submit" disabled={loading}>
            {loading ? '🧙‍♂️ Forging Fiction...' : 'Forge My Fiction!'}
          </button>
        </form>
      </div>

      {loading && <div className="loading-message">Loading... Please wait. This may take several minutes for complex generation options. Grab a cup of tea! ☕</div>}
      
      {!loading && error && <div className="error-message"><strong>Error:</strong> {error}</div>}

      {!loading && responseData && (
        <div className="response-container">
          <h3>Generation Status</h3>
          {responseData.error ? (
             <div className="error-message"><strong>Server Error:</strong> {responseData.error}</div>
          ) : (
            <>
              {/* Display new response fields */}
              <p><strong>Series:</strong> {responseData.series}</p>
              <p><strong>Option Chosen (Backend):</strong> {responseData.option_chosen}</p>
              <p><strong>Model Used:</strong> {responseData.model_used}</p>
            </>
          )}
          
          <h4>Backend Processing Message:</h4>
          <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', maxHeight: '200px', overflowY: 'auto' }}>
            {responseData.message || "No detailed message from backend."}
          </pre>

          {generatedContentParts.length > 0 && generatedContentParts.some(part => part.trim() !== "") && (
            <>
              <hr className="separator" />
              <h4>Generated Content (Story Segment - {generatedContentParts.filter(part => part.trim() !== "").length} part{generatedContentParts.filter(part => part.trim() !== "").length > 1 ? 's' : ''}):</h4>
              {generatedContentParts.map((part, index) => {
                const trimmedPart = part.trim();
                if (trimmedPart === "") return null;
                return (
                  <div key={index}>
                    {generatedContentParts.filter(p => p.trim() !== "").length > 1 && <p><strong>Part {index + 1}:</strong></p>}
                    <pre>{trimmedPart}</pre>
                    {index < generatedContentParts.length - 1 && <hr className="separator" />}
                  </div>
                );
              })}
            </>
          )}
           {responseData.log_identifier && (
            <p className="log-identifier">
              <strong>Log Identifier:</strong> <code>{responseData.log_identifier}</code>
              <br/>
              <em>(Refer to this ID for saved files on the server, typically in a 'jsons/{responseData.log_identifier}' folder)</em>
            </p>
          )}
        </div>
      )}
    </div>
  );
}

export default App;
