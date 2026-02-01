import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import type { OrderPreference, Order } from "../api/client";
import {
  createOrder,
  getOrder,
  getOrders,
  getOrderSummary,
  updateOrder,
} from "../api/client";

const prefs: OrderPreference[] = ["IN_STORE", "DELIVERY", "CURBSIDE"];

const baseSchema = z.object({
  preference: z.enum(["IN_STORE", "DELIVERY", "CURBSIDE"]),
  address: z.string().optional(),
  pickup_time: z.string().optional(),
});

function futureDatetime(s: string) {
  const t = new Date(s);
  return t.getTime() > Date.now();
}

const STEP_PREFERENCE = 1;
const STEP_DETAILS = 2;

export default function Preference() {
  const [step, setStep] = useState(STEP_PREFERENCE);
  const [submitError, setSubmitError] = useState("");
  const [order, setOrder] = useState<Order | null>(null);
  const [orderError, setOrderError] = useState("");
  const [aiSummary, setAiSummary] = useState<{
    summary: string;
    source?: string;
  } | null>(null);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [loading, setLoading] = useState(true);

  const schema = baseSchema.superRefine((data, ctx) => {
    if (data.preference !== "IN_STORE") {
      if (!data.address?.trim()) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["address"],
          message: "Address required",
        });
      }
      if (!data.pickup_time?.trim()) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["pickup_time"],
          message: "Pickup time required",
        });
      } else if (!futureDatetime(data.pickup_time)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["pickup_time"],
          message: "Pickup time must be in the future",
        });
      }
    }
  });

  type FormData = z.infer<typeof schema>;

  const {
    register,
    watch,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { preference: "IN_STORE" },
  });

  const preference = watch("preference");

  useEffect(() => {
    const storedOrderId = localStorage.getItem("orderId");

    if (storedOrderId) {
      getOrder(Number(storedOrderId))
        .then((o) => {
          setOrder(o);
          reset({
            preference: o.preference,
            address: o.address ?? "",
            pickup_time: o.pickup_time ? o.pickup_time.slice(0, 16) : "",
          });
        })
        .catch(() => {
          localStorage.removeItem("orderId");
          setOrderError("");
          getOrders()
            .then((orders) => {
              if (orders.length > 0) {
                const latest = orders[0];
                setOrder(latest);
                localStorage.setItem("orderId", String(latest.id));
                reset({
                  preference: latest.preference,
                  address: latest.address ?? "",
                  pickup_time: latest.pickup_time
                    ? latest.pickup_time.slice(0, 16)
                    : "",
                });
              } else {
                setOrder(null);
              }
            })
            .catch(() => {
              setSubmitError("Failed to load order");
              setOrderError("Failed to load order");
            });
        })
        .finally(() => setLoading(false));
      return;
    }

    getOrders()
      .then((orders) => {
        if (orders.length > 0) {
          const latest = orders[0];
          setOrder(latest);
          localStorage.setItem("orderId", String(latest.id));
          reset({
            preference: latest.preference,
            address: latest.address ?? "",
            pickup_time: latest.pickup_time
              ? latest.pickup_time.slice(0, 16)
              : "",
          });
        } else {
          setOrder(null);
        }
      })
      .catch(() => {
        setOrder(null);
      })
      .finally(() => setLoading(false));
  }, [reset]);

  async function onSubmit(data: FormData) {
    setSubmitError("");
    setOrderError("");
    try {
      const body: {
        preference: OrderPreference;
        address?: string;
        pickup_time?: string;
      } = {
        preference: data.preference,
      };
      if (data.preference !== "IN_STORE") {
        body.address = data.address;
        body.pickup_time = data.pickup_time
          ? new Date(data.pickup_time).toISOString()
          : undefined;
      }
      if (order?.id) {
        await updateOrder(order.id, body);
        const updated = await getOrder(order.id);
        setOrder(updated);
      } else {
        const newOrder = await createOrder(body);
        localStorage.setItem("orderId", String(newOrder.id));
        setOrder(newOrder);
      }
      setStep(STEP_DETAILS);
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : "Failed to save order");
    }
  }

  async function onGenerateSummary() {
    if (!order?.id) return;
    setSummaryLoading(true);
    setAiSummary(null);
    try {
      const data = await getOrderSummary(order.id);
      setAiSummary({ summary: data.summary, source: data.source });
    } catch {
      // Graceful: do nothing
    } finally {
      setSummaryLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="loading">
        <div className="loading-dot" />
      </div>
    );
  }

  return (
    <div className="page page--preference">
      {step === STEP_PREFERENCE ? (
        <>
          {/* <h1 className="page-title page--preference-title">
            Delivery preference
          </h1> */}
          {orderError && (
            <p className="error page--preference-error">{orderError}</p>
          )}

          <div className="preference-layout">
            <div className="preference-col preference-col-form">
              <div className="card preference-card">
                <h2 className="preference-col-heading">Set preference</h2>
                <form onSubmit={handleSubmit(onSubmit)}>
                  <div className="preference-form-stack">
                    <div className="form-group">
                      <label htmlFor="preference" className="label">
                        Preference
                      </label>
                      <select
                        id="preference"
                        {...register("preference")}
                        className="select"
                      >
                        {prefs.map((p) => (
                          <option key={p} value={p}>
                            {p.replace("_", " ")}
                          </option>
                        ))}
                      </select>
                    </div>

                    {(preference === "DELIVERY" ||
                      preference === "CURBSIDE") && (
                      <>
                        <div className="form-group slide-in">
                          <label htmlFor="pickup_time" className="label">
                            Pickup time
                          </label>
                          <input
                            id="pickup_time"
                            type="datetime-local"
                            className="input"
                            {...register("pickup_time")}
                          />
                          {errors.pickup_time && (
                            <p className="error">
                              {errors.pickup_time.message}
                            </p>
                          )}
                        </div>
                        <div className="form-group slide-in">
                          <label htmlFor="address" className="label">
                            Address
                          </label>
                          <input
                            id="address"
                            type="text"
                            className="input"
                            placeholder="Street, city, zip"
                            {...register("address")}
                          />
                          {errors.address && (
                            <p className="error">{errors.address.message}</p>
                          )}
                        </div>
                      </>
                    )}
                  </div>

                  {submitError && <p className="error">{submitError}</p>}
                  <div className="actions preference-actions">
                    <button
                      type="submit"
                      disabled={isSubmitting}
                      className="btn btn-primary "
                    >
                      {isSubmitting ? "Saving..." : "Save"}
                    </button>
                    {order && (
                      <button
                        type="button"
                        onClick={() => setStep(STEP_DETAILS)}
                        className="btn btn-secondary"
                      >
                        Next
                      </button>
                    )}
                  </div>
                </form>
              </div>
            </div>
          </div>
        </>
      ) : (
        <>
          <div className="preference-step-header">
            {/* <h1 className="page-title page--preference-title">
              Delivery details & summary
            </h1> */}
            <button
              type="button"
              onClick={() => setStep(STEP_PREFERENCE)}
              className="btn btn-secondary preference-back-btn"
            >
              Back
            </button>
          </div>
          {orderError && (
            <p className="error page--preference-error">{orderError}</p>
          )}

          <div className="preference-layout">
            <div className="preference-col preference-col-details">
              <div className="card preference-details-card">
                <h2 className="preference-col-heading">Delivery details</h2>
                {order ? (
                  <div className="summary-details">
                    <div className="summary-row">
                      <span className="summary-label">Order</span>
                      <span className="summary-value">#{order.id}</span>
                    </div>
                    <div className="summary-row">
                      <span className="summary-label">Preference</span>
                      <span className="summary-value">
                        {order.preference.replace("_", " ")}
                      </span>
                    </div>
                    {order.address != null && (
                      <div className="summary-row">
                        <span className="summary-label">Address</span>
                        <span className="summary-value summary-value--right">
                          {order.address}
                        </span>
                      </div>
                    )}
                    {order.pickup_time != null && (
                      <div className="summary-row">
                        <span className="summary-label">Pickup time</span>
                        <span className="summary-value">
                          {new Date(order.pickup_time).toLocaleString()}
                        </span>
                      </div>
                    )}
                    <div className="summary-row summary-row--border">
                      <span className="summary-label">Created</span>
                      <span className="summary-value">
                        {new Date(order.created_at).toLocaleString()}
                      </span>
                    </div>
                  </div>
                ) : (
                  <p className="preference-placeholder">
                    No order yet. Go back and save your preference.
                  </p>
                )}
              </div>
            </div>

            <div className="preference-col preference-col-summary">
              <div className="card preference-summary-card">
                <div className="summary-card-header">
                  <h2 className="preference-col-heading">Order summary</h2>
                  {order && (
                    <button
                      type="button"
                      onClick={onGenerateSummary}
                      disabled={summaryLoading}
                      className="btn btn-secondary summary-generate-btn"
                      title="Backend-proxied; uses OpenAI or Gemini when API key is set"
                    >
                      {summaryLoading
                        ? "Generating…"
                        : aiSummary
                        ? "Regenerate"
                        : "Generate AI summary"}
                    </button>
                  )}
                </div>
                {order ? (
                  <>
                    {aiSummary ? (
                      <p className="summary-ai-text">{aiSummary.summary}</p>
                    ) : (
                      <p className="summary-ai-hint">
                        Generate a short AI summary of this order.
                      </p>
                    )}
                    {aiSummary?.source === "ai" && (
                      <span className="summary-ai-badge">
                        Generated with AI
                      </span>
                    )}
                  </>
                ) : (
                  <p className="preference-placeholder">
                    Save your order to generate an AI summary.
                  </p>
                )}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
