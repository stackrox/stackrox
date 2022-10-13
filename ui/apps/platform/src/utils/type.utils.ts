// TODO Comment
export function ensureExhaustive(value: never) {
    // eslint-disable-next-line no-console
    console.error(
        'This should never occur, and if it does there is a mismatch between compile time and runtime values'
    );
    return value;
}
