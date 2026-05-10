package templfiles

import (
	"golang.org/x/text/language"

	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

var (
	MsgInvalidRequest = checkout.LocalizedString{
		language.English: "Invalid request",
		language.Arabic:  "طلب غير صالح",
	}
	MsgInvalidInvoiceID = checkout.LocalizedString{
		language.English: "Invalid invoice ID",
		language.Arabic:  "معرف الفاتورة غير صالح",
	}
	MsgInvoiceNotFound = checkout.LocalizedString{
		language.English: "Invoice not found",
		language.Arabic:  "الفاتورة غير موجودة",
	}
	MsgSessionExpired = checkout.LocalizedString{
		language.English: "Payment session expired",
		language.Arabic:  "انتهت صلاحية جلسة الدفع",
	}
	MsgSessionNotFound = checkout.LocalizedString{
		language.English: "Payment session not found",
		language.Arabic:  "جلسة الدفع غير موجودة",
	}
	MsgInvalidState = checkout.LocalizedString{
		language.English: "Invalid payment state",
		language.Arabic:  "حالة الدفع غير صالحة",
	}
	MsgAlreadyPaid = checkout.LocalizedString{
		language.English: "Invoice already paid",
		language.Arabic:  "الفاتورة مدفوعة بالفعل",
	}
	MsgInternalError = checkout.LocalizedString{
		language.English: "Internal error",
		language.Arabic:  "خطأ داخلي",
	}
	MsgGatewayError = checkout.LocalizedString{
		language.English: "Payment gateway error",
		language.Arabic:  "خطأ في بوابة الدفع",
	}
	MsgPaymentSuccessful = checkout.LocalizedString{
		language.English: "Payment successful",
		language.Arabic:  "تم الدفع بنجاح",
	}
	MsgPaymentDeclined = checkout.LocalizedString{
		language.English: "Your bank declined this transaction",
		language.Arabic:  "رفض البنك هذه المعاملة",
	}
	MsgAuthenticationFailed = checkout.LocalizedString{
		language.English: "Card authentication failed",
		language.Arabic:  "فشل التحقق من البطاقة",
	}
)

var (
	LblSecureCheckout = checkout.LocalizedString{
		language.English: "Secure Checkout",
		language.Arabic:  "الدفع الآمن",
	}
	LblPaymentSuccessful = checkout.LocalizedString{
		language.English: "Payment Successful!",
		language.Arabic:  "تم الدفع بنجاح!",
	}
	LblInvoice = checkout.LocalizedString{
		language.English: "Invoice",
		language.Arabic:  "الفاتورة",
	}
	LblCustomer = checkout.LocalizedString{
		language.English: "Customer",
		language.Arabic:  "العميل",
	}
	LblThankYou = checkout.LocalizedString{
		language.English: "Thank you for your purchase!",
		language.Arabic:  "شكراً لك على الشراء!",
	}
	LblPoweredBy = checkout.LocalizedString{
		language.English: "Powered by Saftaja",
		language.Arabic:  "مدعوم من سفتجة",
	}
	LblAmountDue = checkout.LocalizedString{
		language.English: "Amount Due",
		language.Arabic:  "المبلغ المستحق",
	}
	LblDescription = checkout.LocalizedString{
		language.English: "Description",
		language.Arabic:  "الوصف",
	}
	LblItem = checkout.LocalizedString{
		language.English: "item",
		language.Arabic:  "عنصر",
	}
	LblItems = checkout.LocalizedString{
		language.English: "items",
		language.Arabic:  "عناصر",
	}
	Lbl256BitEncryption = checkout.LocalizedString{
		language.English: "256-bit encryption",
		language.Arabic:  "تشفير 256 بت",
	}
	LblPaymentFailed = checkout.LocalizedString{
		language.English: "Payment Failed",
		language.Arabic:  "فشل الدفع",
	}
	LblPleaseTryAgain = checkout.LocalizedString{
		language.English: "Please try again.",
		language.Arabic:  "يرجى المحاولة مرة أخرى.",
	}
	LblPaymentMethod = checkout.LocalizedString{
		language.English: "Payment Method",
		language.Arabic:  "طريقة الدفع",
	}
	LblNoPaymentMethods = checkout.LocalizedString{
		language.English: "No payment methods available.",
		language.Arabic:  "لا توجد طرق دفع متاحة.",
	}
	LblCreditDebitCard = checkout.LocalizedString{
		language.English: "Credit / Debit Card",
		language.Arabic:  "بطاقة ائتمان / خصم",
	}
	LblVisaMastercardAmex = checkout.LocalizedString{
		language.English: "Visa, Mastercard, Amex",
		language.Arabic:  "فيزا، ماستركارد، أمريكان إكسبريس",
	}
	LblApplePay = checkout.LocalizedString{
		language.English: "Apple Pay",
		language.Arabic:  "أبل باي",
	}
	LblQuickAndSecure = checkout.LocalizedString{
		language.English: "Quick and secure",
		language.Arabic:  "سريع وآمن",
	}
	LblBack = checkout.LocalizedString{
		language.English: "Back",
		language.Arabic:  "رجوع",
	}
	LblCardDetails = checkout.LocalizedString{
		language.English: "Card Details",
		language.Arabic:  "تفاصيل البطاقة",
	}
	LblCardNumber = checkout.LocalizedString{
		language.English: "Card Number",
		language.Arabic:  "رقم البطاقة",
	}
	LblMonth = checkout.LocalizedString{
		language.English: "Month",
		language.Arabic:  "الشهر",
	}
	LblYear = checkout.LocalizedString{
		language.English: "Year",
		language.Arabic:  "السنة",
	}
	LblCVC = checkout.LocalizedString{
		language.English: "CVC",
		language.Arabic:  "رمز الأمان",
	}
	LblCardholderName = checkout.LocalizedString{
		language.English: "Cardholder Name",
		language.Arabic:  "اسم حامل البطاقة",
	}
	LblPay = checkout.LocalizedString{
		language.English: "Pay",
		language.Arabic:  "ادفع",
	}
	LblProcessingPayment = checkout.LocalizedString{
		language.English: "Processing Payment",
		language.Arabic:  "جاري معالجة الدفع",
	}
	LblPleaseWait = checkout.LocalizedString{
		language.English: "Please wait...",
		language.Arabic:  "يرجى الانتظار...",
	}
	LblBankVerification = checkout.LocalizedString{
		language.English: "Bank Verification",
		language.Arabic:  "التحقق البنكي",
	}
	LblConfirmIdentity = checkout.LocalizedString{
		language.English: "Confirm your identity",
		language.Arabic:  "أكّد هويتك",
	}
	LblBankRequired = checkout.LocalizedString{
		language.English: "This verification is required by your bank.",
		language.Arabic:  "هذا التحقق مطلوب من البنك الخاص بك.",
	}
	LblSecuringConnection = checkout.LocalizedString{
		language.English: "Securing connection...",
		language.Arabic:  "جاري تأمين الاتصال...",
	}
	LblDate = checkout.LocalizedString{
		language.English: "Date",
		language.Arabic:  "التاريخ",
	}
	LblPaymentFailedInit = checkout.LocalizedString{
		language.English: "Failed to initialize payment. Please try again.",
		language.Arabic:  "فشل تهيئة الدفع. يرجى المحاولة مرة أخرى.",
	}
	LblGatewayLoadFailed = checkout.LocalizedString{
		language.English: "Payment gateway failed to load. Please refresh.",
		language.Arabic:  "فشل تحميل بوابة الدفع. يرجى تحديث الصفحة.",
	}
	LblCompletingPayment = checkout.LocalizedString{
		language.English: "Completing Payment",
		language.Arabic:  "جاري إتمام الدفع",
	}
	LblClose = checkout.LocalizedString{
		language.English: "Close",
		language.Arabic:  "إغلاق",
	}
	LblTryAgain = checkout.LocalizedString{
		language.English: "Try Again",
		language.Arabic:  "حاول مرة أخرى",
	}
)
