# Phone Mask with Country Code — Booking Form

## Problem

The booking form phone input has no masking, validation, or normalization. Users can type anything and it's sent as-is to the backend. The DB expects E.164-like format (`584243222332`) but nothing enforces it.

## Solution

Auto-detect the tenant's country from `tenants.country` (already in DB as CHAR(2)), show a badge with the country flag + prefix next to the input. User only types local number. Concatenate `prefix + localNumber` before saving.

## Design Decisions

- **Auto-detect by tenant + manual override**: Default country comes from tenant, user can switch via mini-dropdown
- **Badge prefix (not input mask)**: The `+58` appears as a badge/label next to the input, user only types local digits. Simpler than dynamic masks, less confusing.
- **No new libraries**: Only 3 countries supported initially — manual mapping is enough, no need for `libphonenumber-js`
- **Store without `+`**: Phone stored as `584243222332` in `customers.phone` (VARCHAR 20)

## Supported Countries (initial)

| Country | ISO | Prefix | Local Digits | Example |
|---------|-----|--------|-------------|---------|
| Venezuela | VE | +58 | 10 | 4243222332 |
| Brazil | BR | +55 | 10-11 | 41999999999 |
| Dominican Republic | DO | +1 | 10 | 8090000000 |

## UX Flow

```
[ 🇻🇪 +58 ▾ ] [ 4243222332    ]
                 ← numbers only, max digits per country
```

- Tap/click on badge → mini-dropdown with 3 countries (flag + name + prefix)
- Select country → updates prefix badge and digit validation
- Input only accepts numbers (no letters, spaces, or special chars)
- Validation: exact digit count per country (VE=10, DO=10, BR=10-11)

## Changes Required

### Backend (Go)

1. **Expose `country` in public tenant info**: Add `country` field to the response of `GET /api/v1/public/{slug}/info`

### Frontend (Next.js)

1. **Create `PhoneInput` component**:
   - Badge showing flag emoji + prefix + dropdown arrow
   - Number-only input with maxLength based on selected country
   - Mini-dropdown for country selection (3 countries)
   - Props: `defaultCountry`, `value`, `onChange` (returns full number: prefix + local)

2. **Country config map**:
   ```typescript
   const PHONE_COUNTRIES = {
     VE: { prefix: '58', flag: '🇻🇪', name: 'Venezuela', digits: [10] },
     BR: { prefix: '55', flag: '🇧🇷', name: 'Brasil', digits: [10, 11] },
     DO: { prefix: '1', flag: '🇩🇴', name: 'Rep. Dominicana', digits: [10] },
   }
   ```

3. **Validation**: Check digit count matches country config before allowing form submission

4. **Normalization**: Before sending to API, concatenate `prefix + localNumber` (no `+`, no spaces) → e.g. `584243222332`

5. **Integrate in booking form**: Replace current plain `<Input type="tel">` in the contact step with the new `PhoneInput` component. Pass `defaultCountry` from the tenant's `country` field.

### Files Affected

| File | Action | What |
|------|--------|------|
| `apps/api/internal/handler/public.go` | Modify | Add `country` to tenant info response |
| `apps/web/components/ui/PhoneInput.tsx` | Create | Phone input component with country badge |
| `apps/web/app/book/[slug]/page.tsx` | Modify | Use PhoneInput in contact step, pass tenant country |

### Data Flow

```
1. GET /api/v1/public/{slug}/info → { ..., country: "VE" }
2. PhoneInput receives defaultCountry="VE" → shows 🇻🇪 +58 badge
3. User types: 4243222332
4. On submit: PhoneInput.onChange → "584243222332"
5. POST /api/v1/public/{slug}/book → { customer_phone: "584243222332" }
6. Stored in customers.phone as "584243222332"
```
