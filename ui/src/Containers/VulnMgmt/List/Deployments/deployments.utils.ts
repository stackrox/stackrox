export interface Image {
    scan: {
        scanTime: string | null;
    };
}

interface ScanRatio {
    scanned: number;
    total: number;
}

export function getRatioOfScannedImages(images: Image[]): ScanRatio {
    return images.reduce(
        (acc, image) => {
            return image?.scan?.scanTime
                ? { total: acc.total + 1, scanned: acc.scanned + 1 }
                : { ...acc, total: acc.total + 1 };
        },
        { total: 0, scanned: 0 }
    );
}
