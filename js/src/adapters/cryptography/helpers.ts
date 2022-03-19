const ab2str = (buf: ArrayBuffer): string => {
    return String.fromCharCode.apply(null, Array.from(new Uint8Array(buf)));
}

const str2ab = (str: string): Uint8Array => {
    return new TextEncoder().encode(str)
}

const algorithm = "RSA-OAEP"

export {
    ab2str,
    str2ab,
    algorithm,
}