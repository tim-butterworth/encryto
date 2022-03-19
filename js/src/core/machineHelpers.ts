import { BehaviorSubject, Subject, Subscription } from "rxjs";
import { Logger, LoggerKey } from "./logger";

const id = <T>(t: T): T => t;

type TransitionMap<TRANSITIONS extends string, STATE extends string> = {
    [transition in TRANSITIONS]: STATE;
}
type StateNode<T extends string, STATE extends string> = {
    transitions: TransitionMap<T, STATE>,
}

type StateTransitionMap = { [state in string]: string };

type StringMap<T extends object> = {
    [k in keyof T]: k extends string ? k : never
}
type StringKeysOf<T extends object> = StringMap<T>[keyof StringMap<T>]
type TransitionOf<STM extends StateTransitionMap> = STM[keyof STM];
type StateOf<STM extends StateTransitionMap> = StringKeysOf<STM>;

type StateMap<STM extends StateTransitionMap> = { 
    [state in keyof STM]: StateNode<STM[state], StateOf<STM>>;
};

type Machine<STM extends StateTransitionMap, CONTEXT> = {
    pureTransition: (t: TransitionOf<STM>) => void
    transition: (t: TransitionOf<STM>, updateContext: (c: CONTEXT) => CONTEXT) => void,
    subscribe: (fn: (current: { state: keyof STM, context: CONTEXT }) => void) => Subscription
}

type MachineProvider = <STM extends StateTransitionMap, CONTEXT extends object>({
    map, 
    initialState,
} :{
    map: StateMap<STM>, 
    initialState: StateOf<STM>
}) => Machine<STM, CONTEXT>;
const isKeyOfTransitionMap = <O extends object>(key: string | number | symbol, obj: O): key is keyof O => key in obj; 
const newMachine = (logger: Logger): MachineProvider => <STM extends StateTransitionMap, CONTEXT extends object>({
    map, 
    initialState,
} :{
    map: StateMap<STM>, 
    initialState: StateOf<STM>
}): Machine<STM, CONTEXT> => {
    let currentState = initialState;
    let context = {} as CONTEXT;

    const stateSubject = new BehaviorSubject<{ state: keyof STM, context: CONTEXT }>({ state: initialState, context });
    const transitionSubject = new Subject<{ 
        transition: STM[keyof STM],
        updateContext: (context: CONTEXT) => CONTEXT
    }>()

    transitionSubject.subscribe(({ transition, updateContext }) => {
        const currentStateTransitionMap = map[currentState].transitions;
        if (isKeyOfTransitionMap(transition, currentStateTransitionMap)) {
            context = updateContext(context);
            logger(LoggerKey.INFO, `CONTEXT KEYS: [${Object.keys(context)}]`)

            currentState = currentStateTransitionMap[transition]
            stateSubject.next({ 
                state: currentState,
                context
            });
        } else {
            logger(LoggerKey.INFO, `Transition [${transition}] does not exist on state [${currentState}]`)
            logger(LoggerKey.INFO, `Transition info: ${JSON.stringify(currentStateTransitionMap, null, 2)}`)
        }        
    });

    const id = <T>(t: T): T => t;
    return {
        pureTransition: (transition: TransitionOf<STM>) => transitionSubject.next({ transition, updateContext: id }),
        transition: (transition: TransitionOf<STM>, updateContext: (context: CONTEXT) => CONTEXT) => transitionSubject.next({ transition, updateContext }),
        subscribe: (fn: (current: { state: keyof STM, context: CONTEXT }) => void): Subscription => stateSubject.subscribe(fn),
    }
}

export {
    id,
    newMachine
}
export type {
    Machine,
    StateMap,
    StateNode,
    MachineProvider,
}
