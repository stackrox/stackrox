import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import '@testing-library/jest-dom';

import BaseImagesModal from './BaseImagesModal';

const mockMutate = vi.fn();
const mockReset = vi.fn();

const mockUseRestMutation = vi.hoisted(() => vi.fn());

vi.mock('hooks/useRestMutation', () => ({
    default: mockUseRestMutation,
}));

vi.mock('services/BaseImagesService', () => ({
    addBaseImage: vi.fn(),
}));

describe('BaseImagesModal', () => {
    const defaultProps = {
        isOpen: true,
        onClose: vi.fn(),
        onSuccess: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
        mockUseRestMutation.mockReturnValue({
            mutate: mockMutate,
            reset: mockReset,
            isLoading: false,
            isSuccess: false,
            isError: false,
            error: null,
        });
    });

    describe('parseBaseImagePath logic', () => {
        it('should parse simple image path with single colon', async () => {
            render(<BaseImagesModal {...defaultProps} />);

            const input = screen.getByPlaceholderText(/Example: docker.io/);
            const saveButton = screen.getByRole('button', { name: 'Save' });

            fireEvent.change(input, { target: { value: 'ubuntu:22.04' } });
            fireEvent.blur(input);
            fireEvent.click(saveButton);

            await waitFor(() => {
                expect(mockMutate).toHaveBeenCalledWith(
                    {
                        baseImageRepoPath: 'ubuntu',
                        baseImageTagPattern: '22.04',
                    },
                    expect.any(Object)
                );
            });
        });

        it('should parse image path with multiple colons (registry with port)', async () => {
            render(<BaseImagesModal {...defaultProps} />);

            const input = screen.getByPlaceholderText(/Example: docker.io/);
            const saveButton = screen.getByRole('button', { name: 'Save' });

            fireEvent.change(input, {
                target: { value: 'docker.io:5000/library/ubuntu:22.04' },
            });
            fireEvent.blur(input);
            fireEvent.click(saveButton);

            await waitFor(() => {
                expect(mockMutate).toHaveBeenCalledWith(
                    {
                        baseImageRepoPath: 'docker.io:5000/library/ubuntu',
                        baseImageTagPattern: '22.04',
                    },
                    expect.any(Object)
                );
            });
        });

        it('should parse image path with tag pattern', async () => {
            render(<BaseImagesModal {...defaultProps} />);

            const input = screen.getByPlaceholderText(/Example: docker.io/);
            const saveButton = screen.getByRole('button', { name: 'Save' });

            fireEvent.change(input, { target: { value: 'docker.io/library/ubuntu:1.*' } });
            fireEvent.blur(input);
            fireEvent.click(saveButton);

            await waitFor(() => {
                expect(mockMutate).toHaveBeenCalledWith(
                    {
                        baseImageRepoPath: 'docker.io/library/ubuntu',
                        baseImageTagPattern: '1.*',
                    },
                    expect.any(Object)
                );
            });
        });
    });

    describe('form validation', () => {
        it('should show error when field is empty after blur', async () => {
            render(<BaseImagesModal {...defaultProps} />);

            const input = screen.getByPlaceholderText(/Example: docker.io/);

            fireEvent.focus(input);
            fireEvent.blur(input);

            await waitFor(() => {
                expect(screen.getByText('Base image path is required')).toBeInTheDocument();
            });
        });

        it('should enable save button when form is valid', async () => {
            render(<BaseImagesModal {...defaultProps} />);

            const input = screen.getByPlaceholderText(/Example: docker.io/);
            const saveButton = screen.getByRole('button', { name: 'Save' });

            fireEvent.change(input, { target: { value: 'ubuntu:22.04' } });

            await waitFor(() => {
                expect(saveButton).toBeEnabled();
            });
        });
    });

    describe('submission flow', () => {
        it('should show success alert after successful submission', async () => {
            mockUseRestMutation.mockReturnValue({
                mutate: mockMutate,
                reset: mockReset,
                isLoading: false,
                isSuccess: true,
                isError: false,
                error: null,
            });

            render(<BaseImagesModal {...defaultProps} />);

            expect(screen.getByText('Base image successfully added')).toBeInTheDocument();
        });

        it('should call onSuccess callback after successful mutation', async () => {
            const onSuccess = vi.fn();
            render(<BaseImagesModal {...defaultProps} onSuccess={onSuccess} />);

            const input = screen.getByPlaceholderText(/Example: docker.io/);
            const saveButton = screen.getByRole('button', { name: 'Save' });

            fireEvent.change(input, { target: { value: 'ubuntu:22.04' } });
            fireEvent.click(saveButton);

            await waitFor(() => {
                expect(mockMutate).toHaveBeenCalled();
            });

            // Simulate successful mutation by calling the onSuccess callback
            const mutateCall = mockMutate.mock.calls[0];
            const mutationCallbacks = mutateCall[1];

            await act(async () => {
                mutationCallbacks.onSuccess();
            });

            expect(onSuccess).toHaveBeenCalled();
        });
    });
});
