import { BehaviorSubject, Subject, Observable } from "rxjs";
import { Maybe, maybeFns } from "../maybe";

type AppState = {
    encryption: Maybe<{ encryptionKey: string }>;
};

const initialState: AppState = {
    encryption: maybeFns.none(),
};

const getStore = (overrides: Partial<AppState> = {}): { 
    state: Observable<AppState>, 
    dispatch: Subject<{}> 
} => {
    const dispatch = new Subject<{}>();
    const state = new BehaviorSubject<AppState>({
        ...initialState,
        ...overrides,
    });

    dispatch.subscribe((message: {}) => {
        console.log("HI FROM THE STORE!!!!")
        console.log(JSON.stringify(message, null, 2))

        state.next({
            encryption: maybeFns.none(),
        })
    });

    return {
        dispatch,
        state,
    }
};

export { getStore }