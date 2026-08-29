import {useEffect, useState} from 'react';

/** Shows loading UI only after delayMs to avoid flicker on fast requests. */
export function useDebouncedLoading(loading: boolean, delayMs = 150): boolean {
    const [show, setShow] = useState(false);

    useEffect(() => {
        if (!loading) {
            setShow(false);
            return;
        }
        const timer = window.setTimeout(() => setShow(true), delayMs);
        return () => window.clearTimeout(timer);
    }, [loading, delayMs]);

    return show;
}
