import { Encryption, EncryptionProvider } from "../../core/handshake/workflow";
import { algorithm } from "./helpers";

const getEncryption: EncryptionProvider = (key: number[]): Promise<Encryption> => {
  return window.crypto.subtle
    .importKey(
        "spki", 
        new Uint8Array(key), 
        {
        name:"RSA-OAEP", 
        hash: {
            name:"SHA-256"
        }
        }, 
        true, 
        ["encrypt"]
    )
    .then((serverPublicKey) => {
        return {
            encrypt: (message: string): Promise<number[]> => window.crypto.subtle
                .encrypt(
                    algorithm,
                    serverPublicKey,
                    new TextEncoder().encode(message)
                )
                .then((encrypted) => {
                    const byteArray = new Uint8Array(encrypted);
                    return Array.from(byteArray);
                })
        }
    });
};

export {
    getEncryption
}