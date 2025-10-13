import { useState } from 'react';

export default function useMap<K, V>(initialMap: Map<K, V> = new Map()) {
    const [itemMap, setItemMap] = useState(initialMap);

    function has(key: K): boolean {
        return itemMap.has(key);
    }

    function get(key: K): V | undefined {
        return itemMap.get(key);
    }

    function set(key: K, value: V) {
        setItemMap((prevMap) => {
            const nextMap = new Map(prevMap);
            nextMap.set(key, value);
            return nextMap;
        });
    }

    function remove(key: K) {
        setItemMap((prevMap) => {
            const nextMap = new Map(prevMap);
            nextMap.delete(key);
            return nextMap;
        });
    }

    function clear() {
        setItemMap(new Map());
    }

    function values() {
        return itemMap.values();
    }

    return { has, get, set, remove, clear, size: itemMap.size, values };
}
