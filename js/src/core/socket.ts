import { Observable } from "rxjs"

type OutgoingMessage = {}
type IncomingMessage = object;

enum ConnectionStates {
    CONNECTED = "CONNECTED",
    DISCONNECTED = "DISCONNECTED",
    PENDING = "PENDING"
}
type ConnectionStateVariant<T extends ConnectionStates> = {
    connectionVariant: T
};
type Connected = ConnectionStateVariant<ConnectionStates.CONNECTED>;
type Disconnected = ConnectionStateVariant<ConnectionStates.DISCONNECTED>;
type Pending = ConnectionStateVariant<ConnectionStates.PENDING>;
type ConnectionState = Connected | Disconnected | Pending;

const connection = {
    connected: (): Connected => ({
        connectionVariant: ConnectionStates.CONNECTED,
    }),
    disconnected: (): Disconnected => ({
        connectionVariant: ConnectionStates.DISCONNECTED,
    }),
    pending: (): Pending => ({
        connectionVariant: ConnectionStates.PENDING,
    }),
    isConnected: (state: ConnectionState): state is Connected => state.connectionVariant === ConnectionStates.CONNECTED,
    isDisconnected: (state: ConnectionState): state is Disconnected => state.connectionVariant === ConnectionStates.DISCONNECTED,
    isPending: (state: ConnectionState): state is Pending => state.connectionVariant === ConnectionStates.PENDING,
}

type SocketWrapper = {
    incoming: Observable<IncomingMessage>
    dispatch: {
        next: (message: OutgoingMessage) => void
    }
    connectionState: Observable<ConnectionState>
}

export { connection }
export type {
    SocketWrapper,
    OutgoingMessage,
    IncomingMessage,
    ConnectionState,
    Connected,
    Disconnected,
}