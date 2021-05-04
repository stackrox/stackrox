# operator-sdk clone

This directory contains cloned source code of the operator-sdk.  
Helm operator in operator-sdk is internal and not exported. Therefore, we could not reuse helm operator as a library.

For now, we copied operator-sdk at the following commit hash: `c48d4ddad1b90dc17f0736bb1372371a2704f975`.  
Link: <https://github.com/operator-framework/operator-sdk/tree/c48d4ddad1b90dc17f0736bb1372371a2704f975/internal/helm>

The code might diverge in the future and/or we also might decide to take some different approach.
