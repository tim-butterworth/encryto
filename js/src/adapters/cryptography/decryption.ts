import { Decryption } from "../../core/handshake/workflow";
import {
    algorithm,
    ab2str
} from "./helpers"

const generateParams: RsaHashedKeyGenParams = {
    'hash': 'SHA-256',
    'publicExponent': new Uint8Array([1, 0, 1]),
    'name': algorithm,
    "modulusLength": 4096,
};

const getDecryption = (): Promise<Decryption> => {
    return window.crypto.subtle.generateKey(
      generateParams,
      true,
      ["encrypt", "decrypt"]
    )
    .then(({publicKey, privateKey}) => {
        if (publicKey) {
            return window.crypto.subtle.exportKey("spki", publicKey)
              .then((exportedPublic: ArrayBuffer) => ({
                  privateKey,
                  publicKey,
                  rawExportedPublic: exportedPublic
              }));
        } else {
            return Promise.reject();
        }
    })
    .then((keys): Decryption | Promise<never> => {
        const { rawExportedPublic, privateKey } = keys;
    
        const asString = ab2str(rawExportedPublic);
        const base64Encoded = window.btoa(asString);
    
        const pem = `-----BEGIN RSA PUBLIC KEY-----\n${base64Encoded}\n-----END RSA PUBLIC KEY-----`;
    
        console.log("PEM", pem);
    
        if (privateKey) {
            return {
                getPem: () => pem,
                decrypt: (message: number[]): Promise<string> => window.crypto.subtle
                    .decrypt(algorithm, privateKey, new Uint8Array(message))
                    .then(ab2str),
            }
        } else {
            return Promise.reject();
        }
    })
};

export {
    getDecryption
}