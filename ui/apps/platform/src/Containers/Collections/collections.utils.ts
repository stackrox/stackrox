export type CollectionPageAction =
    | {
          type: 'create';
      }
    | {
          type: 'edit' | 'clone' | 'view';
          collectionId: string;
      };

/**
 * Parses raw URL parameters into a valid page action for the collections form.
 *
 * This ensures that page cannot get into invalid states, such as loading a collection ID
 * with 'action=create', or having 'action={edit,clone,view}' without a collection ID.
 *
 * @param action The URL action search parameter
 * @param collectionId The collection ID passed in the URL path
 *
 * @returns A valid collection page action object, or null if the provided parameters are invalid
 */
export function parsePageActionProp(
    action: unknown,
    collectionId: string | undefined
): CollectionPageAction | null {
    if (typeof collectionId === 'undefined' && action === 'create') {
        return { type: 'create' };
    }

    if (typeof collectionId !== 'undefined' && typeof action === 'undefined') {
        return { collectionId, type: 'view' };
    }

    if (
        typeof collectionId !== 'undefined' &&
        (action === 'clone' || action === 'edit' || action === 'view')
    ) {
        return { collectionId, type: action };
    }

    return null;
}
