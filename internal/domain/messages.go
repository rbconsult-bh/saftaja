package domain

import "golang.org/x/text/language"

var (
	MsgInvalidInvoiceID = LocalizedString{
		language.English: "Invalid invoice ID",
		language.Arabic:  "معرف الفاتورة غير صالح",
	}
	MsgInvoiceNotFound = LocalizedString{
		language.English: "Invoice not found",
		language.Arabic:  "الفاتورة غير موجودة",
	}
	MsgSessionExpired = LocalizedString{
		language.English: "Payment session expired",
		language.Arabic:  "انتهت صلاحية جلسة الدفع",
	}
	MsgSessionNotFound = LocalizedString{
		language.English: "Payment session not found",
		language.Arabic:  "جلسة الدفع غير موجودة",
	}
	MsgInvalidState = LocalizedString{
		language.English: "Invalid payment state",
		language.Arabic:  "حالة الدفع غير صالحة",
	}
	MsgAlreadyPaid = LocalizedString{
		language.English: "Invoice already paid",
		language.Arabic:  "الفاتورة مدفوعة بالفعل",
	}
	MsgInternalError = LocalizedString{
		language.English: "Internal error",
		language.Arabic:  "خطأ داخلي",
	}
	MsgGatewayError = LocalizedString{
		language.English: "Payment gateway error",
		language.Arabic:  "خطأ في بوابة الدفع",
	}
	MsgPaymentSuccessful = LocalizedString{
		language.English: "Payment successful",
		language.Arabic:  "تم الدفع بنجاح",
	}
	MsgPaymentDeclined = LocalizedString{
		language.English: "Your bank declined this transaction",
		language.Arabic:  "رفض البنك هذه المعاملة",
	}
	MsgAuthenticationFailed = LocalizedString{
		language.English: "Card authentication failed",
		language.Arabic:  "فشل التحقق من البطاقة",
	}
)

var (
	LblSecureCheckout = LocalizedString{
		language.English: "Secure Checkout",
		language.Arabic:  "الدفع الآمن",
	}
	LblPaymentSuccessful = LocalizedString{
		language.English: "Payment Successful!",
		language.Arabic:  "تم الدفع بنجاح!",
	}
	LblInvoice = LocalizedString{
		language.English: "Invoice",
		language.Arabic:  "الفاتورة",
	}
	LblCustomer = LocalizedString{
		language.English: "Customer",
		language.Arabic:  "العميل",
	}
	LblThankYou = LocalizedString{
		language.English: "Thank you for your purchase!",
		language.Arabic:  "شكراً لك على الشراء!",
	}
	LblPoweredBy = LocalizedString{
		language.English: "Powered by Saftaja",
		language.Arabic:  "مدعوم من سفتجة",
	}
	LblAmountDue = LocalizedString{
		language.English: "Amount Due",
		language.Arabic:  "المبلغ المستحق",
	}
	LblDescription = LocalizedString{
		language.English: "Description",
		language.Arabic:  "الوصف",
	}
	LblItem = LocalizedString{
		language.English: "item",
		language.Arabic:  "عنصر",
	}
	LblItems = LocalizedString{
		language.English: "items",
		language.Arabic:  "عناصر",
	}
	Lbl256BitEncryption = LocalizedString{
		language.English: "256-bit encryption",
		language.Arabic:  "تشفير 256 بت",
	}
	LblPaymentFailed = LocalizedString{
		language.English: "Payment Failed",
		language.Arabic:  "فشل الدفع",
	}
	LblPleaseTryAgain = LocalizedString{
		language.English: "Please try again.",
		language.Arabic:  "يرجى المحاولة مرة أخرى.",
	}
	LblPaymentMethod = LocalizedString{
		language.English: "Payment Method",
		language.Arabic:  "طريقة الدفع",
	}
	LblNoPaymentMethods = LocalizedString{
		language.English: "No payment methods available.",
		language.Arabic:  "لا توجد طرق دفع متاحة.",
	}
	LblCreditDebitCard = LocalizedString{
		language.English: "Credit / Debit Card",
		language.Arabic:  "بطاقة ائتمان / خصم",
	}
	LblVisaMastercardAmex = LocalizedString{
		language.English: "Visa, Mastercard, Amex",
		language.Arabic:  "فيزا، ماستركارد، أمريكان إكسبريس",
	}
	LblApplePay = LocalizedString{
		language.English: "Apple Pay",
		language.Arabic:  "أبل باي",
	}
	LblQuickAndSecure = LocalizedString{
		language.English: "Quick and secure",
		language.Arabic:  "سريع وآمن",
	}
	LblBack = LocalizedString{
		language.English: "Back",
		language.Arabic:  "رجوع",
	}
	LblCardDetails = LocalizedString{
		language.English: "Card Details",
		language.Arabic:  "تفاصيل البطاقة",
	}
	LblCardNumber = LocalizedString{
		language.English: "Card Number",
		language.Arabic:  "رقم البطاقة",
	}
	LblMonth = LocalizedString{
		language.English: "Month",
		language.Arabic:  "الشهر",
	}
	LblYear = LocalizedString{
		language.English: "Year",
		language.Arabic:  "السنة",
	}
	LblCVC = LocalizedString{
		language.English: "CVC",
		language.Arabic:  "رمز الأمان",
	}
	LblCardholderName = LocalizedString{
		language.English: "Cardholder Name",
		language.Arabic:  "اسم حامل البطاقة",
	}
	LblPay = LocalizedString{
		language.English: "Pay",
		language.Arabic:  "ادفع",
	}
	LblProcessingPayment = LocalizedString{
		language.English: "Processing Payment",
		language.Arabic:  "جاري معالجة الدفع",
	}
	LblPleaseWait = LocalizedString{
		language.English: "Please wait...",
		language.Arabic:  "يرجى الانتظار...",
	}
	LblBankVerification = LocalizedString{
		language.English: "Bank Verification",
		language.Arabic:  "التحقق البنكي",
	}
	LblConfirmIdentity = LocalizedString{
		language.English: "Confirm your identity",
		language.Arabic:  "أكّد هويتك",
	}
	LblBankRequired = LocalizedString{
		language.English: "This verification is required by your bank.",
		language.Arabic:  "هذا التحقق مطلوب من البنك الخاص بك.",
	}
	LblSecuringConnection = LocalizedString{
		language.English: "Securing connection...",
		language.Arabic:  "جاري تأمين الاتصال...",
	}
	LblDate = LocalizedString{
		language.English: "Date",
		language.Arabic:  "التاريخ",
	}
	LblPaymentFailedInit = LocalizedString{
		language.English: "Failed to initialize payment. Please try again.",
		language.Arabic:  "فشل تهيئة الدفع. يرجى المحاولة مرة أخرى.",
	}
	LblGatewayLoadFailed = LocalizedString{
		language.English: "Payment gateway failed to load. Please refresh.",
		language.Arabic:  "فشل تحميل بوابة الدفع. يرجى تحديث الصفحة.",
	}
	LblCompletingPayment = LocalizedString{
		language.English: "Completing Payment",
		language.Arabic:  "جاري إتمام الدفع",
	}
	LblClose = LocalizedString{
		language.English: "Close",
		language.Arabic:  "إغلاق",
	}
	LblTryAgain = LocalizedString{
		language.English: "Try Again",
		language.Arabic:  "حاول مرة أخرى",
	}
)
