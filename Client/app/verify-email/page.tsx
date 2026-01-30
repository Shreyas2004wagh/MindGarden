'use client';

import { useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { createClient } from '@/lib/supabase/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { toast } from 'sonner';

export default function VerifyEmailPage() {
    const [otp, setOtp] = useState('');
    const [loading, setLoading] = useState(false);
    const [resending, setResending] = useState(false);
    const searchParams = useSearchParams();
    const email = searchParams.get('email');
    const router = useRouter();

    const handleVerify = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!email) {
            toast.error('Email not found. Please try registering again.');
            router.push('/register');
            return;
        }

        setLoading(true);

        try {
            const supabase = createClient();
            const { error } = await supabase.auth.verifyOtp({
                email: email,
                token: otp,
                type: 'signup',
            });

            if (error) throw error;

            toast.success('Email verified successfully!');
            router.push('/journals');
            router.refresh();
        } catch (err: any) {
            toast.error(err.message || 'Invalid OTP. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    const handleResendOtp = async () => {
        if (!email) {
            toast.error('Email not found. Please try registering again.');
            return;
        }

        setResending(true);

        try {
            const supabase = createClient();
            const { error } = await supabase.auth.resend({
                type: 'signup',
                email: email,
            });

            if (error) throw error;
            toast.success('OTP resent to your email');
        } catch (err: any) {
            toast.error(err.message || 'Failed to resend OTP');
        } finally {
            setResending(false);
        }
    };

    if (!email) {
        return (
            <div className="flex min-h-screen items-center justify-center bg-background px-4">
                <div className="w-full max-w-sm space-y-8 text-center">
                    <h1 className="text-3xl tracking-tight text-foreground">Invalid Request</h1>
                    <p className="text-sm text-muted-foreground">
                        Email not found. Please try registering again.
                    </p>
                    <Button onClick={() => router.push('/register')}>
                        Back to Register
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className="flex min-h-screen items-center justify-center bg-background px-4">
            <div className="w-full max-w-sm space-y-8">
                <div className="space-y-2 text-center">
                    <h1 className="text-3xl tracking-tight text-foreground">Verify Your Email</h1>
                    <p className="text-sm text-muted-foreground">
                        Enter the 8-digit code sent to
                    </p>
                    <p className="text-sm font-medium text-foreground">{email}</p>
                </div>

                <form onSubmit={handleVerify} className="space-y-6">
                    <div className="space-y-2">
                        <Input
                            type="text"
                            placeholder="00000000"
                            value={otp}
                            onChange={(e) => setOtp(e.target.value.replace(/\D/g, ''))}
                            maxLength={8}
                            className="text-center text-2xl tracking-widest"
                            required
                            disabled={loading}
                            autoFocus
                        />
                        <p className="text-xs text-muted-foreground text-center">
                            Enter the 8-digit verification code
                        </p>
                    </div>

                    <Button
                        type="submit"
                        className="w-full"
                        disabled={otp.length !== 8 || loading}
                    >
                        {loading ? 'Verifying...' : 'Verify Email'}
                    </Button>

                    <div className="text-center">
                        <button
                            type="button"
                            onClick={handleResendOtp}
                            disabled={resending}
                            className="text-sm text-primary hover:underline disabled:opacity-50"
                        >
                            {resending ? 'Resending...' : "Didn't receive the code? Resend"}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
