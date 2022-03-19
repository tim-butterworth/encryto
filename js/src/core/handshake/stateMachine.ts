import { Logger } from "../logger";
import { Machine, newMachine, StateMap } from "../machineHelpers";

enum State {
    PENDING = "PENDING",
    ERROR = "ERROR",
    CONNECTED = "CONNECTED",
    CLIENT_KEY_SENT = "KEY_SENT",
    CLIENT_KEY_RECEIVED = "CLIENT_KEY_RECEIVED",
    SERVER_KEY_REQUESTED = "KEY_REQUESTED",
    KEYS_EXCHANGED = "KEYS_EXCHANGED",
    SERVER_VERIFICATION_REQUESTED = "VERIFICATION_REQUESTED",
    SERVER_VERIFICATION_RECEIVED = "VERIFICATION_RECEIVED",
    VERIFICATION_RESPONSE_SENT = "VERIFICATION_RESPONSE_SENT",
    VERIFICATION_CONFIRMED = "VERIFICATION_CONFIRMED"
}

enum Transition {
    CONNECT = "CONNECT",
    ERROR = "ERROR",
    SENT_KEY = "SENT_KEY",
    CONFIRMED_KEY = "CONFIRMED_KEY",
    REQUESTED_SERVER_KEY = "REQUESTED_SERVER_KEY",
    RECEIVED_VERIFICATION = "RECEIVED_VERIFICATION",
    SENT_VERIFICATION = "SENT_VERIFICATION",
    CONFIRM_VERIFICATION = "CONFIRM_VERIFICATION",
    SERVER_KEY_RECEIVED = "SERVER_KEY_RECEIVED",
    REQUEST_VERIFICATION = "REQUEST_VERIFICATION",
    RESET = "RESET"
}

type StateTransitionMap = {
    [State.PENDING]: Transition.CONNECT | Transition.ERROR,
    [State.ERROR]: Transition.ERROR | Transition.RESET,
    [State.CONNECTED]: Transition.ERROR | Transition.SENT_KEY,
    [State.CLIENT_KEY_SENT]: Transition.ERROR | Transition.CONFIRMED_KEY,
    [State.CLIENT_KEY_RECEIVED]: Transition.ERROR | Transition.REQUESTED_SERVER_KEY,
    [State.SERVER_KEY_REQUESTED]: Transition.ERROR | Transition.SERVER_KEY_RECEIVED,
    [State.KEYS_EXCHANGED]: Transition.ERROR | Transition.REQUEST_VERIFICATION,
    [State.SERVER_VERIFICATION_REQUESTED]: Transition.ERROR | Transition.RECEIVED_VERIFICATION,
    [State.SERVER_VERIFICATION_RECEIVED]: Transition.ERROR | Transition.SENT_VERIFICATION,
    [State.VERIFICATION_RESPONSE_SENT]: Transition.ERROR | Transition.CONFIRM_VERIFICATION,
    [State.VERIFICATION_CONFIRMED]: Transition.ERROR,
};

const stateMap: StateMap<StateTransitionMap> = {
    [State.PENDING]: {
        transitions: {
            [Transition.CONNECT]: State.CONNECTED,
            [Transition.ERROR]: State.ERROR
        },
    },
    [State.ERROR]: {
        transitions: {
            [Transition.ERROR]: State.ERROR,
            [Transition.RESET]: State.PENDING,
        },
    },
    [State.CONNECTED]: {
        transitions: {
            [Transition.SENT_KEY]: State.CLIENT_KEY_SENT,
            [Transition.ERROR]: State.ERROR,
        }
    },
    [State.CLIENT_KEY_SENT]: {
        transitions: {
            [Transition.CONFIRMED_KEY]: State.CLIENT_KEY_RECEIVED,
            [Transition.ERROR]: State.ERROR
        }
    },
    [State.CLIENT_KEY_RECEIVED]: {
        transitions: {
            [Transition.REQUESTED_SERVER_KEY]: State.SERVER_KEY_REQUESTED,
            [Transition.ERROR]: State.ERROR
        }
    },
    [State.SERVER_KEY_REQUESTED]: {
        transitions: {
            [Transition.SERVER_KEY_RECEIVED]: State.KEYS_EXCHANGED,
            [Transition.ERROR]: State.ERROR
        }
    },
    [State.KEYS_EXCHANGED]: {
        transitions: {
            [Transition.REQUEST_VERIFICATION]: State.SERVER_VERIFICATION_REQUESTED,
            [Transition.ERROR]: State.ERROR
        }
    },
    [State.SERVER_VERIFICATION_REQUESTED]: {
        transitions: {
            [Transition.RECEIVED_VERIFICATION]: State.SERVER_VERIFICATION_RECEIVED,
            [Transition.ERROR]: State.ERROR
        }
    },
    [State.SERVER_VERIFICATION_RECEIVED]: {
        transitions: {
            [Transition.SENT_VERIFICATION]: State.VERIFICATION_RESPONSE_SENT,
            [Transition.ERROR]: State.ERROR
        }
    },
    [State.VERIFICATION_RESPONSE_SENT]: {
        transitions: {
            [Transition.CONFIRM_VERIFICATION]: State.VERIFICATION_CONFIRMED,
            [Transition.ERROR]: State.ERROR
        }
    },
    [State.VERIFICATION_CONFIRMED]: {
        transitions: {
            [Transition.ERROR]: State.ERROR
        }
    }
}

type HandshakeMachine<CONTEXT extends object> = Machine<StateTransitionMap, CONTEXT>;
const getHandshakeWorkflowMachine = <CONTEXT extends object>(logger: Logger): HandshakeMachine<CONTEXT> => newMachine(logger)<StateTransitionMap, CONTEXT>({ 
    map: stateMap, 
    initialState: State.PENDING
});

export { getHandshakeWorkflowMachine, Transition, State };
export type { HandshakeMachine }
