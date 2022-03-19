import { BehaviorSubject, Subject } from "rxjs";
import { environment } from "../environment";
import {
    SocketWrapper,
    OutgoingMessage,
    IncomingMessage,
    ConnectionState,
    connection
} from "../core/socket";

const getSocketWrapper = (): SocketWrapper => {
    const dispatch = new Subject<OutgoingMessage>();
    const incoming = new Subject<IncomingMessage>();
    const connectionState = new BehaviorSubject<ConnectionState>(connection.pending());

    const ws = new WebSocket(environment.baseUrl);
    ws.onmessage = (message: MessageEvent<any>) => {
        console.log("Websocket Message", message);
        const {data} = message;

        try {
            const json = JSON.parse(data);
            incoming.next(json);
        } catch (e) {
            console.log("Error -> ", e)
        }
    };
    ws.onerror = () => {
        console.log("onerror")
        connectionState.next(connection.disconnected())
    }
    ws.onclose = () => {
        console.log("onclose")
        connectionState.next(connection.disconnected())
    }
    ws.onopen = () => {
        console.log("onopen")
        connectionState.next(connection.connected())
    }

    dispatch.subscribe((outgoing: OutgoingMessage) => {
        ws.send(JSON.stringify(outgoing))
    });

    return {
        dispatch,
        incoming,
        connectionState
    }
}

export { getSocketWrapper };
