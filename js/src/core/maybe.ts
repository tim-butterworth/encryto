enum Variants {
    JUST = 'JUST',
    NONE = 'NONE',
}

type MaybeVariant<T extends Variants> = {
    variant: T
}

type Just<T> = MaybeVariant<Variants.JUST> & { value: T };
type None = MaybeVariant<Variants.NONE>;

type Maybe<T> = Just<T> | None;

const maybeFns = {
    isJust: <T>(maybe: Maybe<T>): maybe is Just<T> => maybe.variant === Variants.JUST,
    isNone: <T>(maybe: Maybe<T>): maybe is None => maybe.variant === Variants.NONE,

    just: <T>(value: T): Just<T> => ({
        variant: Variants.JUST,
        value
    }),
    none: (): None => ({
        variant: Variants.NONE,
    })
}

export { Maybe, maybeFns }