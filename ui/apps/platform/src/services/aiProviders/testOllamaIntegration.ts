/**
 * Integration test script for Ollama provider
 * Run with: npx tsx src/services/aiProviders/testOllamaIntegration.ts
 */

import { createOllamaProvider } from './ollamaProvider';

async function testOllamaIntegration() {
    console.log('🧪 Testing Ollama Provider Integration\n');

    const provider = createOllamaProvider({
        url: 'http://localhost:11434',
        model: 'gemma3:1b', // Use smaller model for faster testing
        timeout: 15000,
    });

    // Test 1: Check availability
    console.log('Test 1: Checking if Ollama is available...');
    try {
        const isAvailable = await provider.isAvailable();
        console.log(`✅ Ollama availability: ${isAvailable ? 'ONLINE' : 'OFFLINE'}\n`);

        if (!isAvailable) {
            console.error('❌ Ollama is not running. Please start Ollama first.');
            process.exit(1);
        }
    } catch (error) {
        console.error('❌ Error checking availability:', error);
        process.exit(1);
    }

    // Test 2: Simple math question
    console.log('Test 2: Testing simple completion (math question)...');
    try {
        const startTime = Date.now();
        const result = await provider.generateCompletion('What is 2+2? Answer with just the number.');
        const duration = Date.now() - startTime;

        console.log(`✅ Response received in ${duration}ms`);
        console.log(`📝 Content: "${result.content}"`);
        console.log(`📊 Confidence: ${result.confidence ?? 'not provided'}`);
        console.log(`💭 Reasoning: ${result.reasoning ?? 'not provided'}\n`);
    } catch (error) {
        console.error('❌ Error generating completion:', error);
        process.exit(1);
    }

    // Test 3: More complex prompt
    console.log('Test 3: Testing complex completion (structured data)...');
    try {
        const startTime = Date.now();
        const prompt = `Convert this to JSON with just the data, no explanation:
Name: John Doe
Age: 30
City: New York`;

        const result = await provider.generateCompletion(prompt);
        const duration = Date.now() - startTime;

        console.log(`✅ Response received in ${duration}ms`);
        console.log(`📝 Content:\n${result.content}\n`);

        // Try parsing as JSON to verify
        try {
            const parsed = JSON.parse(result.content);
            console.log('✅ Successfully parsed as JSON:', parsed);
        } catch {
            console.log('⚠️  Response is not valid JSON (this is okay for this test)');
        }
        console.log();
    } catch (error) {
        console.error('❌ Error generating completion:', error);
        process.exit(1);
    }

    // Test 4: Error handling - empty prompt
    console.log('Test 4: Testing error handling (empty prompt)...');
    try {
        await provider.generateCompletion('');
        console.error('❌ Should have thrown error for empty prompt');
        process.exit(1);
    } catch (error) {
        if (error instanceof Error && error.message.includes('empty')) {
            console.log(`✅ Correctly rejected empty prompt: "${error.message}"\n`);
        } else {
            console.error('❌ Unexpected error:', error);
            process.exit(1);
        }
    }

    console.log('🎉 All integration tests passed!');
    console.log(`\n📊 Provider: ${provider.getName()}`);
}

// Run the tests
testOllamaIntegration().catch((error) => {
    console.error('Fatal error:', error);
    process.exit(1);
});
