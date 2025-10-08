import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { OllamaProvider } from './ollamaProvider';

describe('OllamaProvider', () => {
    let provider: OllamaProvider;

    beforeEach(() => {
        provider = new OllamaProvider({
            url: 'http://localhost:11434',
            model: 'gemma3:1b',
            timeout: 5000,
        });
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe('getName', () => {
        it('should return "Ollama"', () => {
            expect(provider.getName()).toBe('Ollama');
        });
    });

    describe('isAvailable', () => {
        it('should return true when Ollama is reachable', async () => {
            global.fetch = vi.fn().mockResolvedValue({
                ok: true,
            });

            const result = await provider.isAvailable();
            expect(result).toBe(true);
            expect(global.fetch).toHaveBeenCalledWith(
                'http://localhost:11434/api/tags',
                expect.objectContaining({
                    signal: expect.any(AbortSignal),
                })
            );
        });

        it('should return false when Ollama is not reachable', async () => {
            global.fetch = vi.fn().mockRejectedValue(new Error('Connection refused'));

            const result = await provider.isAvailable();
            expect(result).toBe(false);
        });

        it('should return false when request times out', async () => {
            global.fetch = vi.fn().mockImplementation(
                () =>
                    new Promise((resolve) => {
                        setTimeout(resolve, 3000); // Longer than 2s timeout
                    })
            );

            const result = await provider.isAvailable();
            expect(result).toBe(false);
        });
    });

    describe('generateCompletion', () => {
        it('should generate completion successfully', async () => {
            const mockResponse = {
                model: 'gemma3:1b',
                created_at: '2025-10-08T22:00:00Z',
                response: 'This is a test response',
                done: true,
            };

            global.fetch = vi.fn().mockResolvedValue({
                ok: true,
                json: async () => mockResponse,
            });

            const result = await provider.generateCompletion('Test prompt');

            expect(result).toEqual({
                content: 'This is a test response',
            });
            expect(global.fetch).toHaveBeenCalledWith(
                'http://localhost:11434/api/generate',
                expect.objectContaining({
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: expect.stringContaining('Test prompt'),
                })
            );
        });

        it('should trim whitespace from response', async () => {
            const mockResponse = {
                model: 'gemma3:1b',
                created_at: '2025-10-08T22:00:00Z',
                response: '  Response with whitespace  \n',
                done: true,
            };

            global.fetch = vi.fn().mockResolvedValue({
                ok: true,
                json: async () => mockResponse,
            });

            const result = await provider.generateCompletion('Test prompt');
            expect(result.content).toBe('Response with whitespace');
        });

        it('should throw error for empty prompt', async () => {
            await expect(provider.generateCompletion('')).rejects.toThrow(
                'Prompt cannot be empty'
            );
        });

        it('should throw error for whitespace-only prompt', async () => {
            await expect(provider.generateCompletion('   ')).rejects.toThrow(
                'Prompt cannot be empty'
            );
        });

        it('should throw error when Ollama API returns error status', async () => {
            global.fetch = vi.fn().mockResolvedValue({
                ok: false,
                status: 500,
                text: async () => 'Internal server error',
            });

            await expect(provider.generateCompletion('Test prompt')).rejects.toThrow(
                'Ollama API error (500)'
            );
        });

        it('should throw timeout error when request exceeds timeout', async () => {
            global.fetch = vi.fn().mockImplementation(
                (_, options) =>
                    new Promise((_, reject) => {
                        // Simulate abort signal triggering
                        const signal = options?.signal as AbortSignal;
                        if (signal) {
                            signal.addEventListener('abort', () => {
                                const error = new Error('The operation was aborted');
                                error.name = 'AbortError';
                                reject(error);
                            });
                        }
                    })
            );

            await expect(provider.generateCompletion('Test prompt')).rejects.toThrow(
                'Ollama request timed out'
            );
        });

        it('should throw connection error when fetch fails', async () => {
            global.fetch = vi.fn().mockRejectedValue(new Error('fetch failed'));

            await expect(provider.generateCompletion('Test prompt')).rejects.toThrow(
                'Cannot connect to Ollama'
            );
        });

        it('should throw error when response is missing content', async () => {
            global.fetch = vi.fn().mockResolvedValue({
                ok: true,
                json: async () => ({
                    model: 'gemma3:1b',
                    done: true,
                    response: '',
                }),
            });

            await expect(provider.generateCompletion('Test prompt')).rejects.toThrow(
                'Ollama response missing content'
            );
        });
    });
});
