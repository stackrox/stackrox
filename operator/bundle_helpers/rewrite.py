#!/usr/bin/env python3
import logging


def rewrite(d, rewriter):
    """
    Rewrite rewrites a dictionary recursively, by applying rewriter to all elements.

    If rewriter returns something (i.e., not None), it will not traverse further, but instead replace the
    current value with the result.
    """
    res = rewriter(d)
    if res is not None:
        return res
    if isinstance(d, list):
        for i, elem in enumerate(d):
            res = rewrite(elem, rewriter)
            if res is not None:
                logging.info(f'Replaced: {d[i]} with {res}')
                d[i] = res
    if isinstance(d, dict):
        updates = []
        for k, v in d.items():
            res = rewrite(v, rewriter)
            if res is not None:
                updates.append((k, res))
        for k, v in updates:
            logging.info(f'Replaced: {d[k]} with {v}')
            d[k] = v


def string_replacer(old, new):
    """
    Returns a rewrite function that does a literal string match and replace.
    """
    def rewriter(val):
        if isinstance(val, str) and val == old:
            return new
    return rewriter
