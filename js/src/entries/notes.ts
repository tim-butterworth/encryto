import { getStore } from "../core/state/store"
import { Decryption, getHandshakeWorkflow } from "../core/handshake/workflow";
import { getSocketWrapper } from "../adapters/websocketClient";
import { getHandshakeWorkflowMachine } from "../core/handshake/stateMachine";
import { getDecryption } from "../adapters/cryptography/decryption";
import { getEncryption } from "../adapters/cryptography/encryption";
import { getLogger } from "../adapters/logger";
import { Logger, LoggerKey } from "../core/logger";
import { getConnectionMachine } from "../core/connection";
import { newMachine } from "../core/machineHelpers";

const logger: Logger = getLogger();

const store = getStore();
console.log(store);

const setupForm = (dispatch: (m: object) => void) => {
  const appDiv = document.querySelector("#app")
  if (appDiv) {
      const idInput = document.createElement("input");
      idInput.setAttribute("type", "text");
      idInput.setAttribute("placeholder", "id");
  
      const idDiv = document.createElement("div");
      idDiv.appendChild(idInput);
  
      const messageInput = document.createElement("input");
      messageInput.setAttribute("type", "text");
      messageInput.setAttribute("placeholder", "message");
      
      const messageDiv = document.createElement("div");
      messageDiv.appendChild(messageInput);
  
      const button = document.createElement("button");
      button.onclick = () => {
          if (idInput.value != "") {
            dispatch({
              Varient: "Envelope",
              Data: {
                To: idInput.value,
                Data: {
                Varient: "RealMessage",
                  Content: messageInput.value,
                }
              }
            });

            idInput.value = "";
            messageInput.value = "";
          }
      };
      button.innerText = "submit";
      const buttonDiv = document.createElement("div");
      buttonDiv.appendChild(button);
  
      const child = document.createElement("div");
      child.appendChild(idDiv);
      child.appendChild(messageDiv);
      child.appendChild(buttonDiv);
  
      appDiv.appendChild(child);
  
      Object.assign(document, { dispatch });
  } else {
      console.log("ERROR")
  }
}

getDecryption()
  .then((decryption: Decryption) => 
    getConnectionMachine(
      getSocketWrapper,
      getHandshakeWorkflow(
        logger,
        decryption,
        getEncryption,
        () => getHandshakeWorkflowMachine(logger)
      ),
      newMachine(logger)
    )
  )
  .then(({ dispatch, incoming }) => {
    setupForm(dispatch)
    incoming.subscribe((message) => {
      console.log("-------------------FROM INCOMING-------------------");
      console.log(JSON.stringify(message, null, 2));
      console.log("-------------------FROM INCOMING-------------------");
    });
  })
  .catch((e) => {
    logger(LoggerKey.ERROR, `ERROR: ${e}`)
  })
