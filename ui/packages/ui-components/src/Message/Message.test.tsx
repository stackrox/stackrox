import React from 'react';
import { render, screen } from '@testing-library/react';
import { ArrowRight } from 'react-feather';

import Message, { baseClasses } from './Message';

describe('Message', () => {
    it('should render component', () => {
        const testMessage = 'This is a test';

        render(<Message>{testMessage}</Message>);

        const el = screen.getByTestId('message');
        // smoke test
        expect(el).toHaveClass(baseClasses);

        // content
        expect(el).toHaveTextContent(testMessage);

        // should have a default icon
        expect(screen.getByTestId('info-icon')).toHaveClass('h-6 w-6');
    });

    it('should accept children prop', () => {
        const body = (
            <div data-testid="test-body">
                <p>A paragraph</p>
                <p>A second paragraph</p>
            </div>
        );

        render(<Message>{body}</Message>);

        const el = screen.getByTestId('test-body');
        expect(el).toHaveTextContent('A paragraphA second paragraph');
    });

    it('should accept custom icon prop', () => {
        const customIcon = (
            <ArrowRight className="h-8 w-8" strokeWidth="2px" data-testid="arrow-icon" />
        );

        render(<Message icon={customIcon}>A fake body</Message>);

        expect(screen.getByTestId('arrow-icon')).toHaveClass('h-8 w-8');
    });

    it('should accept extra classes for its root element', () => {
        const extraClasses = 'awesome playtpus';

        render(<Message extraClasses={extraClasses}>A fake body</Message>);

        expect(screen.getByTestId('message')).toHaveClass(extraClasses);
    });

    it('should accept extra classes for its body element', () => {
        const extraBodyClasses = 'whomping willow';
        const testMessage = 'This is a test';

        render(<Message extraBodyClasses={extraBodyClasses}>{testMessage}</Message>);

        // content
        expect(screen.getByTestId('message-body')).toHaveClass(extraBodyClasses);
    });

    it('should accept a type of base by default', () => {
        const testMessage = 'This is a test';

        render(<Message>{testMessage}</Message>);

        // content
        expect(screen.getByTestId('message')).toHaveClass('base-message');
    });

    it('should accept a type of success', () => {
        const testMessage = 'This is a test';

        render(<Message type="success">{testMessage}</Message>);

        // content
        expect(screen.getByTestId('message')).toHaveClass('success-message');
    });

    it('should accept a type of warn', () => {
        const testMessage = 'This is a test';

        render(<Message type="warn">{testMessage}</Message>);

        // content
        expect(screen.getByTestId('message')).toHaveClass('warn-message');
    });

    it('should accept a type of error', () => {
        const testMessage = 'This is a test';

        render(<Message type="error">{testMessage}</Message>);

        // content
        expect(screen.getByTestId('message')).toHaveClass('error-message');
    });
});
