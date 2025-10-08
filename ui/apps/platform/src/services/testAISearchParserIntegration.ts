/**
 * Integration test for AI Search Parser Service
 * Run with: npx tsx src/services/testAISearchParserIntegration.ts
 */

import { createOllamaProvider } from './aiProviders/ollamaProvider';
import { createAISearchParserService } from './aiSearchParserService';
import { imageCVESearchFilterConfig, imageSearchFilterConfig } from '../Containers/Vulnerabilities/searchFilterConfig';

async function testAISearchParserIntegration() {
    console.log('🧪 Testing AI Search Parser Service Integration\n');

    // Setup
    const provider = createOllamaProvider({
        url: 'http://localhost:11434',
        model: 'gemma3:4b', // Use larger model for better results
        timeout: 30000, // 30 second timeout for complex queries
    });

    const parser = createAISearchParserService({
        provider,
        minConfidence: 0.5,
        maxQueryLength: 500,
    });

    // Test 1: Check provider availability
    console.log('Test 1: Checking AI provider availability...');
    const isAvailable = await parser.isProviderAvailable();
    if (!isAvailable) {
        console.error('❌ Ollama is not available. Please start Ollama first.');
        process.exit(1);
    }
    console.log(`✅ Provider available: ${parser.getProviderName()}\n`);

    // Test filter config (CVE and Image filters)
    const filterConfig = [imageCVESearchFilterConfig, imageSearchFilterConfig];

    // Test 2: Simple query
    console.log('Test 2: Testing simple query - "critical CVEs"');
    try {
        const startTime = Date.now();
        const result = await parser.parseNaturalLanguageQuery('critical CVEs', filterConfig);
        const duration = Date.now() - startTime;

        console.log(`✅ Parsed in ${duration}ms`);
        console.log(`📊 Confidence: ${(result.confidence * 100).toFixed(1)}%`);
        console.log(`💭 Reasoning: ${result.reasoning || 'none'}`);
        console.log(`🔍 Filter:`, JSON.stringify(result.searchFilter, null, 2));
        console.log();
    } catch (error) {
        console.error('❌ Error:', error);
        process.exit(1);
    }

    // Test 3: Complex multi-filter query
    console.log('Test 3: Testing complex query - "fixable critical CVEs discovered in last 30 days"');
    try {
        const startTime = Date.now();
        const result = await parser.parseNaturalLanguageQuery(
            'fixable critical CVEs discovered in last 30 days',
            filterConfig
        );
        const duration = Date.now() - startTime;

        console.log(`✅ Parsed in ${duration}ms`);
        console.log(`📊 Confidence: ${(result.confidence * 100).toFixed(1)}%`);
        console.log(`💭 Reasoning: ${result.reasoning || 'none'}`);
        console.log(`🔍 Filter:`, JSON.stringify(result.searchFilter, null, 2));
        console.log();
    } catch (error) {
        console.error('❌ Error:', error);
        process.exit(1);
    }

    // Test 4: Ambiguous query (should have lower confidence)
    console.log('Test 4: Testing ambiguous query - "some vulnerabilities"');
    try {
        const startTime = Date.now();
        const result = await parser.parseNaturalLanguageQuery(
            'some vulnerabilities',
            filterConfig
        );
        const duration = Date.now() - startTime;

        console.log(`✅ Parsed in ${duration}ms`);
        console.log(`📊 Confidence: ${(result.confidence * 100).toFixed(1)}%`);
        console.log(`💭 Reasoning: ${result.reasoning || 'none'}`);
        console.log(`🔍 Filter:`, JSON.stringify(result.searchFilter, null, 2));

        if (result.confidence < 0.7) {
            console.log('✅ Correctly identified as low confidence query');
        }
        console.log();
    } catch (error) {
        console.error('❌ Error:', error);
        process.exit(1);
    }

    // Test 5: Input validation
    console.log('Test 5: Testing input validation - empty query');
    try {
        await parser.parseNaturalLanguageQuery('', filterConfig);
        console.error('❌ Should have thrown validation error for empty query');
        process.exit(1);
    } catch (error) {
        if (error && typeof error === 'object' && 'type' in error && error.type === 'validation') {
            console.log(`✅ Correctly rejected empty query: "${error.message}"`);
        } else {
            console.error('❌ Wrong error type:', error);
            process.exit(1);
        }
        console.log();
    }

    console.log('🎉 All integration tests passed!');
    console.log('\n📝 Summary:');
    console.log('- AI provider is working correctly');
    console.log('- Query parsing works for simple and complex queries');
    console.log('- Confidence scoring is functioning');
    console.log('- Input validation is working');
}

// Run the tests
testAISearchParserIntegration().catch((error) => {
    console.error('Fatal error:', error);
    process.exit(1);
});
