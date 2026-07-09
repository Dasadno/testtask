CREATE TABLE subscriptions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name text        NOT NULL,
    price        integer     NOT NULL CHECK (price >= 0),
    user_id      uuid        NOT NULL,
    start_date   date        NOT NULL,
    end_date     date,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT end_date_not_before_start CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions (user_id);
CREATE INDEX idx_subscriptions_service_name ON subscriptions (service_name);
