import type { SourceType as ImageSourceType } from 'types/image.proto';
import type { SourceType as ScanComponentSourceType } from 'types/scanComponent.proto';

export const DEFAULT_VM_PAGE_SIZE = 20;

// Display labels

export type SourceType = ImageSourceType | ScanComponentSourceType;

export const sourceTypes: readonly SourceType[] = [
    'OS',
    'PYTHON',
    'JAVA',
    'RUBY',
    'NODEJS',
    'GO',
    'DOTNETCORERUNTIME',
    'INFRASTRUCTURE',
] as const;

export const sourceTypeLabels: Record<SourceType, string> = Object.freeze({
    OS: 'OS',
    PYTHON: 'Python',
    JAVA: 'Java',
    RUBY: 'Ruby',
    NODEJS: 'Node.js',
    GO: 'Go',
    DOTNETCORERUNTIME: '.NET Core Runtime',
    INFRASTRUCTURE: 'Infrastructure',
});
